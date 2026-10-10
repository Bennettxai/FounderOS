package schemahealth

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Requirement is one runtime schema object the current server code expects.
type Requirement struct {
	Kind   string
	Name   string
	Detail string
}

func (r Requirement) String() string {
	if r.Detail == "" {
		return r.Kind + " " + r.Name
	}
	return r.Kind + " " + r.Name + " (" + r.Detail + ")"
}

// Report is the result of checking a database against the runtime schema
// contract.
type Report struct {
	Missing []Requirement
}

func (r Report) OK() bool {
	return len(r.Missing) == 0
}

func (r Report) Error() string {
	if r.OK() {
		return ""
	}
	items := make([]string, 0, len(r.Missing))
	for _, req := range r.Missing {
		items = append(items, req.String())
	}
	return "database schema is missing required runtime objects: " + strings.Join(items, ", ")
}

// ErrSchemaDrift marks schema contract failures so callers can distinguish DB
// connectivity from a connected but wrong-shaped database.
var ErrSchemaDrift = errors.New("database schema drift")

type SchemaError struct {
	Report Report
}

func (e *SchemaError) Error() string {
	return e.Report.Error()
}

func (e *SchemaError) Unwrap() error {
	return ErrSchemaDrift
}

// Check verifies the current database has the schema required by the current
// BusinessOS runtime. It intentionally checks concrete tables and columns used
// by module handlers, not every historical table in schema.sql.
func Check(ctx context.Context, pool *pgxpool.Pool) (Report, error) {
	if pool == nil {
		return Report{}, fmt.Errorf("schema health check requires a database pool")
	}
	return check(ctx, pool)
}

type schemaQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func check(ctx context.Context, db schemaQuerier) (Report, error) {
	// Fetch the schema once: per-object round trips can exceed the startup
	// deadline when the application and database run in different regions.
	var tables, columns, functions []string
	err := db.QueryRow(ctx, `
		SELECT
			ARRAY(SELECT table_name::text FROM information_schema.tables WHERE table_schema = 'public'),
			ARRAY(SELECT table_name::text || '.' || column_name::text FROM information_schema.columns WHERE table_schema = 'public'),
			ARRAY(SELECT p.proname::text FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace WHERE n.nspname = 'public')
	`).Scan(&tables, &columns, &functions)
	if err != nil {
		return Report{}, err
	}
	tableSet, columnSet, functionSet := nameSet(tables), nameSet(columns), nameSet(functions)

	report := Report{}

	for _, table := range RequiredTables {
		if !tableSet[table] {
			report.Missing = append(report.Missing, Requirement{
				Kind: "table",
				Name: table,
			})
		}
	}

	columnTables := make([]string, 0, len(RequiredColumns))
	for table := range RequiredColumns {
		columnTables = append(columnTables, table)
	}
	sort.Strings(columnTables)
	for _, table := range columnTables {
		for _, column := range RequiredColumns[table] {
			if !columnSet[table+"."+column] {
				report.Missing = append(report.Missing, Requirement{
					Kind:   "column",
					Name:   table + "." + column,
					Detail: "module runtime dependency",
				})
			}
		}
	}

	for _, function := range RequiredFunctions {
		if !functionSet[function] {
			report.Missing = append(report.Missing, Requirement{
				Kind: "function",
				Name: function,
			})
		}
	}

	if !report.OK() {
		return report, &SchemaError{Report: report}
	}

	return report, nil
}

func nameSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[name] = true
	}
	return set
}
