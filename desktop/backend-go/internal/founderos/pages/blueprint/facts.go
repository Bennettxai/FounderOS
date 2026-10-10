package blueprint

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Facts is a node's key → value facts in the order the compiler lists them.
// FounderOS v1 builds them as a JS object, which keeps insertion order, and the
// blueprint renders them in that order (container chips, inspector Facts); a
// Go map would come out alphabetical. Marshals as a plain JSON object.
type Facts struct {
	keys []string
	vals map[string]string
}

// facts builds Facts from alternating key, value pairs.
func facts(kv ...string) Facts {
	var f Facts
	for i := 0; i+1 < len(kv); i += 2 {
		f.Set(kv[i], kv[i+1])
	}
	return f
}

// Set adds a fact at the end, or overwrites one in place.
func (f *Facts) Set(k, v string) {
	if f.vals == nil {
		f.vals = map[string]string{}
	}
	if _, ok := f.vals[k]; !ok {
		f.keys = append(f.keys, k)
	}
	f.vals[k] = v
}

// Get returns a fact's value, "" when absent.
func (f Facts) Get(k string) string { return f.vals[k] }

func (f Facts) Len() int { return len(f.keys) }

// Clone copies the facts so a node can add its own without touching the source.
func (f Facts) Clone() Facts {
	var c Facts
	for _, k := range f.keys {
		c.Set(k, f.vals[k])
	}
	return c
}

func (f Facts) String() string {
	parts := make([]string, len(f.keys))
	for i, k := range f.keys {
		parts[i] = k + ":" + f.vals[k]
	}
	return "{" + strings.Join(parts, " ") + "}"
}

func (f Facts) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range f.keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		vb, err := json.Marshal(f.vals[k])
		if err != nil {
			return nil, err
		}
		b.Write(kb)
		b.WriteByte(':')
		b.Write(vb)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

func (f *Facts) UnmarshalJSON(data []byte) error {
	*f = Facts{}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return fmt.Errorf("facts: want a JSON object")
	}
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return err
		}
		k, _ := kt.(string)
		var v string
		if err := dec.Decode(&v); err != nil {
			return err
		}
		f.Set(k, v)
	}
	_, err := dec.Token()
	return err
}
