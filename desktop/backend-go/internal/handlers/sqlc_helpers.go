package handlers

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Tiny helpers shared across handlers when shaping sqlc-generated rows
// for the EngineSync mixin. Kept in one file so it's clear they're
// generic value coercions, not module-specific logic.

// optString dereferences an optional string column. nil → "".
func optString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// pgTimestampToTime converts a pgtype.Timestamptz to a UTC time.Time,
// returning the zero time when the column is null.
func pgTimestampToTime(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time.UTC()
}

// pgPlainTimestampToTime is the without-timezone equivalent. sqlc generates
// pgtype.Timestamp for `TIMESTAMP WITHOUT TIME ZONE` columns; we coerce
// those to UTC the same way.
func pgPlainTimestampToTime(t pgtype.Timestamp) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time.UTC()
}

// pgDateToTime converts a pgtype.Date to time.Time at midnight UTC,
// returning the zero time when the column is null.
func pgDateToTime(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return d.Time.UTC()
}
