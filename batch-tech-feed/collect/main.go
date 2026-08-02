package main

import (
	"context"
	"fmt"
	"os"

	"github.com/k-wa-wa/pechka/batch-tech-feed/cmd"
)

// 収集を担う Go 側のエントリポイント。台本生成から公開までは Python 側
// (main.py) が担当する。両者は別の Job / 別のイメージで動く。
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: tech-feed <collect> [args]\n")
		os.Exit(1)
	}

	ctx := context.Background()
	cmdName := os.Args[1]
	subArgs := os.Args[2:]

	var err error
	switch cmdName {
	case "collect":
		err = cmd.RunCollect(ctx, subArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nUsage: tech-feed <collect> [args]\n", cmdName)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running %s command: %v\n", cmdName, err)
		os.Exit(1)
	}
}
