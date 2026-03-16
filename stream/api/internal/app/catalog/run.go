package catalog

import (
	"context"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	infraMongo "pechka/streaming-service/api/internal/infrastructure/mongo"
	"pechka/streaming-service/api/internal/infrastructure/postgres"
	apiInterface "pechka/streaming-service/api/internal/interface/api"
	"pechka/streaming-service/api/internal/usecase"
)

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

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		log.Fatal("MONGODB_URI is not set")
	}
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("mongo connect failed: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	mongoDBName := os.Getenv("MONGODB_DB")
	if mongoDBName == "" {
		log.Fatal("MONGODB_DB is not set")
	}
	mongoDB := mongoClient.Database(mongoDBName)



	metaRepo := postgres.NewContentRepository(pgPool)
	catalogRepo := infraMongo.NewCatalogRepository(mongoDB)

	uc := usecase.NewCatalogUseCase(catalogRepo)
	handler := apiInterface.NewCatalogHandler(uc, metaRepo)

	app := fiber.New()
	app.Use(logger.New())

	apiGroup := app.Group("/api/catalog/v1")
	handler.RegisterRoutes(apiGroup)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	log.Printf("Catalog Service starting on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("fiber listen failed: %v", err)
	}
}
