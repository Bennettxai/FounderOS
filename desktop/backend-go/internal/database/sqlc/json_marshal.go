package sqlc

import (
	"encoding/json"
	"strings"
)

// MarshalJSON for NullTaskpriority - returns lowercase string value or null
func (ns NullTaskpriority) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(strings.ToLower(string(ns.Taskpriority)))
}

// MarshalJSON for NullTaskstatus - returns lowercase string value or null
func (ns NullTaskstatus) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(strings.ToLower(string(ns.Taskstatus)))
}

// MarshalJSON for NullProjectstatus - returns lowercase string value or null
func (ns NullProjectstatus) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(strings.ToLower(string(ns.Projectstatus)))
}

// MarshalJSON for NullProjectpriority - returns lowercase string value or null
func (ns NullProjectpriority) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(strings.ToLower(string(ns.Projectpriority)))
}

// MarshalJSON for NullContexttype - returns lowercase string value or null
func (ns NullContexttype) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(strings.ToLower(string(ns.Contexttype)))
}

// MarshalJSON for NullNodetype - returns lowercase string value or null
func (ns NullNodetype) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(strings.ToLower(string(ns.Nodetype)))
}

// MarshalJSON for NullNodehealth - returns lowercase string value or null
func (ns NullNodehealth) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(strings.ToLower(string(ns.Nodehealth)))
}

// MarshalJSON for NullClientstatus - returns lowercase string value or null
func (ns NullClientstatus) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(strings.ToLower(string(ns.Clientstatus)))
}

// MarshalJSON for NullArtifacttype - returns lowercase string value or null
func (ns NullArtifacttype) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(strings.ToLower(string(ns.Artifacttype)))
}

// MarshalJSON for NullInteractiontype - returns lowercase string value or null
func (ns NullInteractiontype) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(strings.ToLower(string(ns.Interactiontype)))
}

// MarshalJSON for NullMemberstatus - returns lowercase string value or null
func (ns NullMemberstatus) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(strings.ToLower(string(ns.Memberstatus)))
}

// MarshalJSON for NullClienttype - returns lowercase string value or null
func (ns NullClienttype) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(strings.ToLower(string(ns.Clienttype)))
}
