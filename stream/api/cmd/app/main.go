package main

import (
	"fmt"
	"os"

	"pechka/streaming-service/api/internal/app/auth"
	"pechka/streaming-service/api/internal/app/catalog"
	"pechka/streaming-service/api/internal/app/importer"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: stream [subcommand]")
		fmt.Println("Subcommands: catalog-service, auth-service, batch-nfs-importer")
		os.Exit(1)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "catalog-service":
		catalog.Run()
	case "auth-service":
		auth.Run()

	case "batch-nfs-importer":
		importer.Run()
	default:
		fmt.Printf("Unknown subcommand: %s\n", subcommand)
		fmt.Println("Subcommands: catalog-service, auth-service, batch-nfs-importer")
		os.Exit(1)
	}
}
