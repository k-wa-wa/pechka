package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/bwmarrin/snowflake"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/k-wa-wa/pechka/api/internal/config"
	"github.com/k-wa-wa/pechka/api/internal/handler"
	apiMiddleware "github.com/k-wa-wa/pechka/api/internal/middleware"
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

	// Initialize Kubernetes Dynamic Client
	var k8sConfig *rest.Config
	k8sConfig, err = rest.InClusterConfig()
	if err != nil {
		slog.Info("in-cluster config not found, falling back to local kubeconfig")
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		k8sConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{}).ClientConfig()
		if err != nil {
			slog.Error("failed to load kubernetes config", "error", err)
			os.Exit(1)
		}
	}
	dynClient, err := dynamic.NewForConfig(k8sConfig)
	if err != nil {
		slog.Error("failed to create kubernetes dynamic client", "error", err)
		os.Exit(1)
	}

	minioClient, err := minio.New(cfg.MinioURL, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		slog.Error("failed to create minio client", "error", err)
		os.Exit(1)
	}

	sfNode, err := snowflake.NewNode(1)
	if err != nil {
		slog.Error("snowflake node init failed", "error", err)
		os.Exit(1)
	}

	pgContent := pgRepo.NewContentRepository(pgPool)
	pgChannel := pgRepo.NewChannelRepository(pgPool)
	pgDisc := pgRepo.NewDiscRepository(pgPool)
	pgSubtitle := pgRepo.NewSubtitleRepository(pgPool)
	mgContent := mongoRepo.NewContentRepository(mongoDB)
	mgSubtitle := mongoRepo.NewSubtitleRepository(mongoDB)
	esContent := elasticRepo.NewContentRepository(esClient)

	contentsH := handler.NewContentsHandler(mgContent)
	channelsH := handler.NewChannelsHandler(pgChannel)
	subtitlesH := handler.NewSubtitlesHandler(mgSubtitle)
	searchH := handler.NewSearchHandler(esContent)
	adminH := handler.NewAdminHandler(pgContent, pgDisc, pgSubtitle, sfNode).WithChannelRepository(pgChannel)
	k8sNamespace := currentNamespace()
	slog.Info("resolved k8s namespace for workflow triggers", "namespace", k8sNamespace)
	ingestH := handler.NewIngestHandler(dynClient, k8sNamespace).WithChannelRepository(pgChannel)
	uploadH := handler.NewUploadHandler(pgContent, minioClient, cfg.MinioBucket, dynClient, sfNode, k8sNamespace).WithChannelRepository(pgChannel)

	e := setupRouter(RouterParams{
		AllowedIPRange:   cfg.AllowedIPRange,
		ContentsHandler:  contentsH,
		ChannelsHandler:  channelsH,
		SubtitlesHandler: subtitlesH,
		SearchHandler:    searchH,
		AdminHandler:     adminH,
		IngestHandler:    ingestH,
		UploadHandler:    uploadH,
	})

	slog.Info("starting server", "port", cfg.Port)
	if err := e.Start(":" + cfg.Port); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

// RouterParams holds dependencies and configuration for setting up the Echo router.
type RouterParams struct {
	AllowedIPRange    string
	MetricsRegisterer prometheus.Registerer
	MetricsGatherer   prometheus.Gatherer
	ContentsHandler   *handler.ContentsHandler
	ChannelsHandler   *handler.ChannelsHandler
	SubtitlesHandler  *handler.SubtitlesHandler
	SearchHandler     *handler.SearchHandler
	AdminHandler      *handler.AdminHandler
	IngestHandler     *handler.IngestHandler
	UploadHandler     *handler.UploadHandler
}

// setupRouter configures middleware and registers routes on an Echo instance.
func setupRouter(p RouterParams) *echo.Echo {
	if p.ChannelsHandler == nil {
		panic("setupRouter: ChannelsHandler is required")
	}
	if p.ContentsHandler == nil {
		panic("setupRouter: ContentsHandler is required")
	}
	if p.SubtitlesHandler == nil {
		panic("setupRouter: SubtitlesHandler is required")
	}
	if p.SearchHandler == nil {
		panic("setupRouter: SearchHandler is required")
	}
	if p.IngestHandler == nil {
		panic("setupRouter: IngestHandler is required")
	}
	if p.AdminHandler == nil {
		panic("setupRouter: AdminHandler is required")
	}
	if p.UploadHandler == nil {
		panic("setupRouter: UploadHandler is required")
	}

	e := echo.New()
	e.HideBanner = true
	e.IPExtractor = echo.ExtractIPFromRealIPHeader()

	reg := p.MetricsRegisterer
	gath := p.MetricsGatherer
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	if gath == nil {
		gath = prometheus.DefaultGatherer
	}

	e.Use(echoprometheus.NewMiddlewareWithConfig(echoprometheus.MiddlewareConfig{
		Subsystem:  "pechka_api",
		Registerer: reg,
	}))
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
			}
			// 404/400のようなクライアント起因のエラーは正常系の一部として扱いINFOに留める。
			// サーバー起因(5xx、またはHTTPError以外の予期しないエラー)のみERRORとする。
			if v.Status >= 500 {
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
	e.GET("/metrics", echoprometheus.NewHandlerWithConfig(echoprometheus.HandlerConfig{
		Gatherer: gath,
	}))

	v1 := e.Group("/api/v1")

	v1.GET("/channels", p.ChannelsHandler.List)
	v1.GET("/channels/:id", p.ChannelsHandler.Get)
	v1.GET("/contents", p.ContentsHandler.List)
	v1.GET("/contents/:short_id", p.ContentsHandler.Get)
	v1.GET("/contents/:short_id/variants", p.ContentsHandler.GetVariants)
	v1.GET("/contents/:short_id/subtitles/:lang", p.SubtitlesHandler.GetVTT)
	v1.GET("/search", p.SearchHandler.Search)
	v1.POST("/contents/ingest", p.IngestHandler.TriggerIngest, apiMiddleware.IPFilter(p.AllowedIPRange))

	admin := v1.Group("/admin")
	// admin配下は元々IP制限が掛かっていなかった(ingestのみ個別適用)。
	// 字幕admin CRUD追加を機に、admin全体へ一括で適用する。
	admin.Use(apiMiddleware.IPFilter(p.AllowedIPRange))
	admin.GET("/contents", p.AdminHandler.ListContents)
	admin.POST("/contents", p.AdminHandler.CreateContent)
	admin.POST("/contents/upload", p.UploadHandler.UploadVideo)
	admin.PUT("/contents/:id", p.AdminHandler.UpdateContent)
	admin.DELETE("/contents/:id", p.AdminHandler.DeleteContent)
	admin.POST("/contents/:id/archive", p.AdminHandler.ArchiveContent)
	admin.POST("/contents/:id/unarchive", p.AdminHandler.UnarchiveContent)
	admin.GET("/discs", p.AdminHandler.ListDiscs)
	admin.POST("/discs", p.AdminHandler.CreateDisc)
	admin.GET("/contents/:content_id/subtitles", p.AdminHandler.ListSubtitleTracks)
	admin.PUT("/subtitles/:track_id/status", p.AdminHandler.UpdateSubtitleTrackStatus)
	admin.GET("/subtitles/:track_id/cues", p.AdminHandler.ListSubtitleCues)
	admin.POST("/subtitles/:track_id/cues", p.AdminHandler.InsertSubtitleCue)
	admin.PUT("/subtitles/cues/:cue_id", p.AdminHandler.UpdateSubtitleCue)
	admin.DELETE("/subtitles/cues/:cue_id", p.AdminHandler.DeleteSubtitleCue)

	return e
}

// currentNamespace は Argo Workflow を起動する先の namespace を返す。
// prod/previewでnamespace名が異なる(pechka / pechka-pr-<N>)ため、この値を
// ハードコードすると一方の環境でRBAC上作成が拒否される。in-cluster実行時は
// kubeletがServiceAccountごとに自動マウントするnamespaceファイルから読む。
func currentNamespace() string {
	const nsFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	if data, err := os.ReadFile(nsFile); err == nil {
		if ns := strings.TrimSpace(string(data)); ns != "" {
			return ns
		}
	}
	return "pechka"
}
