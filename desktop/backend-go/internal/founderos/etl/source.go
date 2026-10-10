package etl

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// sourceDB is one read-only SQLite file.
type sourceDB struct {
	path   string
	db     *sql.DB
	tables map[string]bool
}

// readOnlyDSN opens a file without ever writing to it. A file with a WAL (or
// a hot rollback journal) is opened mode=ro so SQLite still reads the log; a
// file without one is also marked immutable, which lets SQLite open it even
// where it could not create a -shm next to it.
func readOnlyDSN(path string) string {
	dsn := "file:" + (&url.URL{Path: path}).EscapedPath() + "?mode=ro"
	_, walErr := os.Stat(path + "-wal")
	_, jErr := os.Stat(path + "-journal")
	if os.IsNotExist(walErr) && os.IsNotExist(jErr) {
		dsn += "&immutable=1"
	}
	return dsn
}

func openSource(path string) (*sourceDB, error) {
	db, err := sql.Open("sqlite", readOnlyDSN(path))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table'`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("open %s read-only: %w", path, err)
	}
	defer rows.Close()
	s := &sourceDB{path: path, db: db, tables: map[string]bool{}}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			db.Close()
			return nil, err
		}
		s.tables[n] = true
	}
	return s, rows.Err()
}

func (s *sourceDB) Close() error { return s.db.Close() }

func (s *sourceDB) count(table string) (int, error) {
	var n int
	err := s.db.QueryRow(fmt.Sprintf(`SELECT count(*) FROM %q`, table)).Scan(&n)
	return n, err
}

// columns lists a table's columns.
func (s *sourceDB) columns(table string) (map[string]bool, error) {
	rows, err := s.db.Query(fmt.Sprintf(`SELECT name FROM pragma_table_info(%s)`, quoteLit(table)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out[n] = true
	}
	return out, rows.Err()
}

// readAll returns every row in rowid order, each as column -> value, with the
// rowid under rowidCol.
func (s *sourceDB) readAll(table string) ([]map[string]any, error) {
	rows, err := s.db.Query(fmt.Sprintf(`SELECT rowid AS %s, * FROM %q ORDER BY rowid`, rowidCol, table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		m := make(map[string]any, len(cols))
		for i, cname := range cols {
			if b, ok := vals[i].([]byte); ok {
				m[cname] = string(b)
				continue
			}
			m[cname] = vals[i]
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func quoteLit(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// locateSources finds the source files in dir. founderos-os.db is required; the
// others are optional. state.db is looked for at <dir>/state.db, then
// <dir>/slack-bridge/state.db.
func locateSources(dir string) (map[string]string, error) {
	found := map[string]string{}
	for _, f := range []string{FileOS, FileBank, FileLedger, FilePaykit} {
		p := filepath.Join(dir, f)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			found[f] = p
		}
	}
	for _, p := range []string{filepath.Join(dir, FileState), filepath.Join(dir, "slack-bridge", FileState)} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			found[FileState] = p
			break
		}
	}
	if _, ok := found[FileOS]; !ok {
		return nil, fmt.Errorf("%s not found in %s (it is the one required source file)", FileOS, dir)
	}
	return found, nil
}
