package main

import (
	"context"
	"log"
	"os"

	"github.com/bwmarrin/snowflake"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/k-wa-wa/pechka/api/internal/handler"
	elasticRepo "github.com/k-wa-wa/pechka/api/internal/repository/elastic"
	mongoRepo "github.com/k-wa-wa/pechka/api/internal/repository/mongo"
	pgRepo "github.com/k-wa-wa/pechka/api/internal/repository/postgres"
)

func main() {
	ctx := context.Background()

	pgPool, err := pgxpool.New(ctx, mustEnv("POSTGRES_DSN"))
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pgPool.Close()

	mongoClient, err := mongo.Connect(options.Client().ApplyURI(mustEnv("MONGO_URL")))
	if err != nil {
		log.Fatalf("mongo: %v", err)
	}
	defer mongoClient.Disconnect(ctx)
	mongoDB := mongoClient.Database(getEnv("MONGO_DB", "stream"))

	esClient, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{mustEnv("ELASTICSEARCH_URL")},
	})
	if err != nil {
		log.Fatalf("elasticsearch: %v", err)
	}

	sfNode, err := snowflake.NewNode(1)
	if err != nil {
		log.Fatalf("snowflake: %v", err)
	}

	pgContent := pgRepo.NewContentRepository(pgPool)
	pgDisc := pgRepo.NewDiscRepository(pgPool)
	mgContent := mongoRepo.NewContentRepository(mongoDB)
	esContent := elasticRepo.NewContentRepository(esClient)

	catalogH := handler.NewCatalogHandler(mgContent)
	searchH := handler.NewSearchHandler(esContent)
	adminH := handler.NewAdminHandler(pgContent, pgDisc, sfNode)

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	v1 := e.Group("/v1")

	v1.GET("/catalog", catalogH.List)
	v1.GET("/catalog/:short_id", catalogH.Get)
	v1.GET("/catalog/:short_id/variants", catalogH.GetVariants)
	v1.GET("/search", searchH.Search)

	admin := v1.Group("/admin")
	admin.GET("/contents", adminH.ListContents)
	admin.POST("/contents", adminH.CreateContent)
	admin.PUT("/contents/:id", adminH.UpdateContent)
	admin.DELETE("/contents/:id", adminH.DeleteContent)
	admin.GET("/discs", adminH.ListDiscs)
	admin.POST("/discs", adminH.CreateDisc)

	port := getEnv("PORT", "8080")
	log.Fatal(e.Start(":" + port))
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env %s is not set", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
