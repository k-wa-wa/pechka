package main

import (
	"fmt"
	"os"

	"pechka/streaming-service/api/internal/app/auth"
	"pechka/streaming-service/api/internal/app/catalog"
	"pechka/streaming-service/api/internal/app/importer"
	"pechka/streaming-service/api/internal/app/metadata"
	"pechka/streaming-service/api/internal/app/sync"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: stream [subcommand]")
		fmt.Println("Subcommands: metadata-service, catalog-service, auth-service, sync-batch, batch-nfs-importer")
		os.Exit(1)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "metadata-service":
		metadata.Run()
	case "catalog-service":
		catalog.Run()
	case "auth-service":
		auth.Run()
	case "sync-batch":
		sync.Run()
	case "batch-nfs-importer":
		importer.Run()
	default:
		fmt.Printf("Unknown subcommand: %s\n", subcommand)
		fmt.Println("Subcommands: metadata-service, catalog-service, auth-service, sync-batch, batch-nfs-importer")
		os.Exit(1)
	}
}
