package metadata

import (
	"context"
	"log"
	"os"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	apiInterface "pechka/streaming-service/api/internal/interface/api"
	"pechka/streaming-service/api/internal/infrastructure/idgen"
	"pechka/streaming-service/api/internal/infrastructure/postgres"
	"pechka/streaming-service/api/internal/usecase"
)

func getEnvInt64(key string, defaultVal int64) int64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	var i int64
	fmt.Sscanf(val, "%d", &i)
	return i
}

func Run() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}
	log.Println("Connected to PostgreSQL successfully")

	nodeID := getEnvInt64("NODE_ID", 1)
	idGen := idgen.NewSnowflakeGenerator(nodeID)

	repo := postgres.NewContentRepository(pool)
	uc := usecase.NewContentUseCase(repo, idGen)
	handler := apiInterface.NewContentHandler(uc)

	app := fiber.New()
	app.Use(logger.New())

	apiGroup := app.Group("/api/metadata/v1")
	handler.RegisterRoutes(apiGroup)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Metadata Service starting on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
