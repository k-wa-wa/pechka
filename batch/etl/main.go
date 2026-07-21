package main

import (
	"context"
	"fmt"
	"os"

	"github.com/k-wa-wa/pechka/batch/etl/cmd"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: pechka-etl <extract|transform|load> [args]\n")
		os.Exit(1)
	}

	ctx := context.Background()
	cmdName := os.Args[1]
	subArgs := os.Args[2:]

	var err error
	switch cmdName {
	case "extract":
		err = cmd.RunExtract(ctx, subArgs)
	case "transform":
		err = cmd.RunTransform(ctx, subArgs)
	case "load":
		err = cmd.RunLoad(ctx, subArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nUsage: pechka-etl <extract|transform|load> [args]\n", cmdName)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running %s command: %v\n", cmdName, err)
		os.Exit(1)
	}
}
