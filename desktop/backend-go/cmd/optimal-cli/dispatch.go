package main

import "fmt"

// dispatch routes a command name to its handler. Adding a new subcommand
// is a single line in this map plus a function in commands.go — keeps the
// switch from sprawling.
var commands = map[string]func([]string, globalOpts) error{
	"health":    cmdHealth,
	"stats":     cmdStats,
	"status":    cmdStatus,
	"ls":        cmdLs,
	"search":    cmdSearch,
	"graph":     cmdGraph,
	"hubs":      cmdHubs,
	"knowledge": cmdKnowledge,
	"ingest":    cmdIngest,
}

func dispatch(name string, args []string, opts globalOpts) error {
	fn, ok := commands[name]
	if !ok {
		return fmt.Errorf("unknown command %q — run optimal-cli help", name)
	}
	return fn(args, opts)
}
