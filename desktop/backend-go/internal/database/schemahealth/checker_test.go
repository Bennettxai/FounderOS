package schemahealth

import (
	"errors"
	"testing"
)

func TestReportErrorListsMissingObjects(t *testing.T) {
	report := Report{
		Missing: []Requirement{
			{Kind: "table", Name: "workspace_apps"},
			{Kind: "column", Name: "deals.client_id", Detail: "module runtime dependency"},
		},
	}

	got := report.Error()
	want := "database schema is missing required runtime objects: table workspace_apps, column deals.client_id (module runtime dependency)"
	if got != want {
		t.Fatalf("unexpected error\nwant: %s\n got: %s", want, got)
	}
}

func TestSchemaErrorWrapsSchemaDriftSentinel(t *testing.T) {
	err := &SchemaError{Report: Report{Missing: []Requirement{{Kind: "table", Name: "workspaces"}}}}
	if !errors.Is(err, ErrSchemaDrift) {
		t.Fatalf("expected SchemaError to wrap ErrSchemaDrift")
	}
}
