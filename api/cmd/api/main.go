package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/bwmarrin/snowflake"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/k-wa-wa/pechka/api/internal/config"
	"github.com/k-wa-wa/pechka/api/internal/handler"
	elasticRepo "github.com/k-wa-wa/pechka/api/internal/repository/elastic"
	mongoRepo "github.com/k-wa-wa/pechka/api/internal/repository/mongo"
	pgRepo "github.com/k-wa-wa/pechka/api/internal/repository/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()
	ctx := context.Background()

	pgPool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		slog.Error("postgres connection failed", "error", err)
		os.Exit(1)
	}
	defer pgPool.Close()

	mongoClient, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURL))
	if err != nil {
		slog.Error("mongo connection failed", "error", err)
		os.Exit(1)
	}
	defer mongoClient.Disconnect(ctx)
	mongoDB := mongoClient.Database(cfg.MongoDB)

	esClient, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{cfg.ElasticsearchURL},
	})
	if err != nil {
		slog.Error("elasticsearch connection failed", "error", err)
		os.Exit(1)
	}

	sfNode, err := snowflake.NewNode(1)
	if err != nil {
		slog.Error("snowflake node init failed", "error", err)
		os.Exit(1)
	}

	pgContent := pgRepo.NewContentRepository(pgPool)
	pgDisc := pgRepo.NewDiscRepository(pgPool)
	mgContent := mongoRepo.NewContentRepository(mongoDB)
	esContent := elasticRepo.NewContentRepository(esClient)

	contentsH := handler.NewContentsHandler(mgContent)
	searchH := handler.NewSearchHandler(esContent)
	adminH := handler.NewAdminHandler(pgContent, pgDisc, sfNode)

	e := echo.New()
	e.HideBanner = true
	e.Use(echoprometheus.NewMiddleware("pechka_api"))
	e.Use(middleware.RequestID())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogMethod:   true,
		LogLatency:  true,
		LogRemoteIP: true,
		LogError:    true,
		HandleError: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			attrs := []any{
				"method", v.Method,
				"uri", v.URI,
				"status", v.Status,
				"latency_ms", v.Latency.Milliseconds(),
				"remote_ip", v.RemoteIP,
				"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
			}
			if v.Error != nil {
				attrs = append(attrs, "error", v.Error)
				slog.ErrorContext(c.Request().Context(), "request", attrs...)
			} else {
				slog.InfoContext(c.Request().Context(), "request", attrs...)
			}
			return nil
		},
	}))
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	e.GET("/metrics", echoprometheus.NewHandler())

	v1 := e.Group("/v1")

	v1.GET("/contents", contentsH.List)
	v1.GET("/contents/:short_id", contentsH.Get)
	v1.GET("/contents/:short_id/variants", contentsH.GetVariants)
	v1.GET("/search", searchH.Search)

	admin := v1.Group("/admin")
	admin.GET("/contents", adminH.ListContents)
	admin.POST("/contents", adminH.CreateContent)
	admin.PUT("/contents/:id", adminH.UpdateContent)
	admin.DELETE("/contents/:id", adminH.DeleteContent)
	admin.GET("/discs", adminH.ListDiscs)
	admin.POST("/discs", adminH.CreateDisc)

	slog.Info("starting server", "port", cfg.Port)
	if err := e.Start(":" + cfg.Port); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
