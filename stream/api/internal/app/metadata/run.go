package metadata

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"pechka/streaming-service/api/internal/infrastructure/idgen"
	"pechka/streaming-service/api/internal/infrastructure/postgres"
	apiInterface "pechka/streaming-service/api/internal/interface/api"
	"pechka/streaming-service/api/internal/interface/api/middleware"
	"pechka/streaming-service/api/internal/usecase"
)

func getEnvInt64(key string, defaultVal int64) int64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	var i int64
	_, _ = fmt.Sscanf(val, "%d", &i)
	return i
}

func Run() {
	_ = godotenv.Load()

	pgURL := os.Getenv("DATABASE_URL")
	if pgURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}
	pgPool, err := pgxpool.New(context.Background(), pgURL)
	if err != nil {
		log.Fatalf("pg connect failed: %v", err)
	}
	defer pgPool.Close()

	contentRepo := postgres.NewContentRepository(pgPool)
	userRepo := postgres.NewUserRepository(pgPool)
	nodeID := getEnvInt64("NODE_ID", 2)
	idGen := idgen.NewSnowflakeGenerator(nodeID)

	contentUC := usecase.NewContentUseCase(contentRepo, idGen)
	handler := apiInterface.NewMetadataHandler(contentUC, userRepo)

	app := fiber.New()
	app.Use(logger.New())

	apiGroup := app.Group("/api/metadata/v1")
	apiGroup.Use(middleware.RequireAppJWT())
	apiGroup.Use(middleware.RequirePermission("content", "write"))

	handler.RegisterRoutes(apiGroup)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}
	log.Printf("Metadata Service starting on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("fiber listen failed: %v", err)
	}
}
