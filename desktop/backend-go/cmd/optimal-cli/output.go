package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// printPretty emits any value as 2-indented JSON on stdout. We use json
// marshaling instead of a custom table renderer so the output is always
// machine-readable; operators can pipe through `| jq` or `| less` without
// the CLI making formatting choices.
func printPretty(v any) {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "format: %v\n", err)
		return
	}
	fmt.Println(string(out))
}

// newLineScanner is split out so it's easy to swap (e.g. bufio with a
// larger buffer) without changing every call site. Cookie files are tiny
// today; this future-proofs against bigger jars without a rewrite.
func newLineScanner(r io.Reader) *bufio.Scanner {
	return bufio.NewScanner(r)
}
