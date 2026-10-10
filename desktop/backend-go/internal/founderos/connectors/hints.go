package connectors

import "strings"

// EnvFileHint is where an operator puts credentials by hand (DefaultResolver's
// EnvLocal). The board's API keys section plants them instead, and the
// process env works too.
const EnvFileHint = "~/.founderos/.env"

// SetKeys is the not-configured hint for a connector that needs keys:
// "Set A and B in ~/.founderos/.env or under API keys."
func SetKeys(keys ...string) string {
	list := strings.Join(keys, ", ")
	if n := len(keys); n > 1 {
		list = strings.Join(keys[:n-1], ", ") + " and " + keys[n-1]
	}
	return "Set " + list + " in " + EnvFileHint + " or under API keys."
}

// NotFound says a key resolved nowhere: "KEY not found in env or ~/.founderos/.env."
func NotFound(key string) string {
	return key + " not found in env or " + EnvFileHint + "."
}
