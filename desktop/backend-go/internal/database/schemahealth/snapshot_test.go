package schemahealth

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

type snapshotDB struct {
	calls                      int
	tables, columns, functions []string
	err                        error
}

func (db *snapshotDB) QueryRow(ctx context.Context, _ string, _ ...any) pgx.Row {
	db.calls++
	if err := ctx.Err(); err != nil {
		return snapshotRow{err: err}
	}
	return snapshotRow{db: db, err: db.err}
}

type snapshotRow struct {
	db  *snapshotDB
	err error
}

func (row snapshotRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	*dest[0].(*[]string) = row.db.tables
	*dest[1].(*[]string) = row.db.columns
	*dest[2].(*[]string) = row.db.functions
	return nil
}

func completeSnapshot() *snapshotDB {
	db := &snapshotDB{tables: RequiredTables, functions: RequiredFunctions}
	for table, columns := range RequiredColumns {
		for _, column := range columns {
			db.columns = append(db.columns, table+"."+column)
		}
	}
	return db
}

func TestCheckUsesOneDatabaseRoundTrip(t *testing.T) {
	db := completeSnapshot()
	report, err := check(context.Background(), db)
	if err != nil || !report.OK() {
		t.Fatalf("report=%v error=%v", report, err)
	}
	if db.calls != 1 {
		t.Fatalf("schema check made %d database round trips, want 1", db.calls)
	}
}

func TestSnapshotStillRejectsMissingObjects(t *testing.T) {
	db := completeSnapshot()
	db.tables = db.tables[1:]
	db.columns = db.columns[1:]
	db.functions = nil
	report, err := check(context.Background(), db)
	if !errors.Is(err, ErrSchemaDrift) {
		t.Fatalf("expected schema drift, got %v", err)
	}
	if len(report.Missing) != 3 {
		t.Fatalf("expected missing table, column and function, got %v", report.Missing)
	}
}

func TestSnapshotPreservesDatabaseErrors(t *testing.T) {
	want := errors.New("database unavailable")
	_, err := check(context.Background(), &snapshotDB{err: want})
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

func TestSnapshotHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := check(ctx, completeSnapshot())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want cancellation", err)
	}
}
