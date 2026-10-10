package etl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigration161IsMirroredInSchemaSQLAndAllowsOE(t *testing.T) {
	mig, err := os.ReadFile(filepath.Join(dbDir(), "migrations", "161_founderos_oe_ingest.sql"))
	if err != nil {
		t.Fatal(err)
	}
	schema, _ := os.ReadFile(filepath.Join(dbDir(), "schema.sql"))
	if !strings.Contains(string(schema), strings.TrimSpace(string(mig))) {
		t.Fatal("schema.sql must contain migrations/161_founderos_oe_ingest.sql verbatim")
	}
	for _, want := range []string{"'oe'", "founderos_call_archive"} {
		if !strings.Contains(string(mig), want) {
			t.Errorf("migration 161 must mention %s", want)
		}
	}
}

func TestMigration162IsMirroredInSchemaSQL(t *testing.T) {
	mig, err := os.ReadFile(filepath.Join(dbDir(), "migrations", "162_founderos_device_pushes.sql"))
	if err != nil {
		t.Fatal(err)
	}
	schema, _ := os.ReadFile(filepath.Join(dbDir(), "schema.sql"))
	if !strings.Contains(string(schema), strings.TrimSpace(string(mig))) {
		t.Fatal("schema.sql must contain migrations/162_founderos_device_pushes.sql verbatim")
	}
}

func TestMigration163IsMirroredInSchemaSQLAndKeepsPaykitDatesRaw(t *testing.T) {
	mig, err := os.ReadFile(filepath.Join(dbDir(), "migrations", "163_founderos_paykit_raw_dates.sql"))
	if err != nil {
		t.Fatal(err)
	}
	schema, _ := os.ReadFile(filepath.Join(dbDir(), "schema.sql"))
	if !strings.Contains(string(schema), strings.TrimSpace(string(mig))) {
		t.Fatal("schema.sql must contain migrations/163_founderos_paykit_raw_dates.sql verbatim")
	}
	if !strings.Contains(string(mig), "last_transaction_date TYPE TEXT") {
		t.Error("migration 163 must make last_transaction_date TEXT")
	}
}
