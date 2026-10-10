package google

import (
	"encoding/json"
	"io"
)

// decodeJSON decodes JSON from a reader into the target.
func decodeJSON(r io.Reader, target interface{}) error {
	return json.NewDecoder(r).Decode(target)
}
