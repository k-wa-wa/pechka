package main

import (
	"context"
	"fmt"
	"os"

	"github.com/k-wa-wa/pechka/batch-tech-feed/produce/cmd"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: produce <synthesize|render|publish|build|all> [args]\n")
		os.Exit(1)
	}

	ctx := context.Background()
	cmdName := os.Args[1]
	subArgs := os.Args[2:]

	var err error
	switch cmdName {
	case "synthesize":
		_, err = cmd.RunSynthesize(ctx, subArgs)
	case "render":
		err = cmd.RunRender(ctx, subArgs)
	case "publish":
		err = cmd.RunPublish(ctx, subArgs)
	case "build":
		_, err = cmd.RunSynthesize(ctx, subArgs)
		if err == nil {
			err = cmd.RunRender(ctx, subArgs)
		}
	case "all":
		_, err = cmd.RunSynthesize(ctx, subArgs)
		if err == nil {
			err = cmd.RunRender(ctx, subArgs)
		}
		if err == nil {
			err = cmd.RunPublish(ctx, subArgs)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nUsage: produce <synthesize|render|publish|build|all> [args]\n", cmdName)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running %s command: %v\n", cmdName, err)
		os.Exit(1)
	}
}
