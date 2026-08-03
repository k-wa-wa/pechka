package main

import (
	"context"
	"fmt"
	"os"

	"github.com/k-wa-wa/pechka/batch-tech-feed/cmd"
)

// 収集・選定・一次情報検証・台本生成を担う Go 側のエントリポイント。
// 音声合成から動画化・配信までは Python 側/別Job (produce) が担当する。
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: tech-feed <pipeline|collect|filter|enrich|compose> [args]\n")
		os.Exit(1)
	}

	ctx := context.Background()
	cmdName := os.Args[1]
	subArgs := os.Args[2:]

	var err error
	switch cmdName {
	case "pipeline":
		err = cmd.RunPipeline(ctx, subArgs)
	case "collect":
		err = cmd.RunCollect(ctx, subArgs)
	case "filter":
		err = cmd.RunFilter(ctx, subArgs)
	case "enrich":
		err = cmd.RunEnrich(ctx, subArgs)
	case "compose":
		err = cmd.RunCompose(ctx, subArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nUsage: tech-feed <pipeline|collect|filter|enrich|compose> [args]\n", cmdName)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running %s command: %v\n", cmdName, err)
		os.Exit(1)
	}
}
