package sync

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	infraMongo "pechka/streaming-service/api/internal/infrastructure/mongo"
	"pechka/streaming-service/api/internal/infrastructure/postgres"
	infraRedis "pechka/streaming-service/api/internal/infrastructure/redis"
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

	if err := pgPool.Ping(context.Background()); err != nil {
		log.Fatalf("pg ping failed: %v", err)
	}

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

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("REDIS_URL is not set")
	}
	redisClient := redis.NewClient(&redis.Options{Addr: redisURL})
	defer redisClient.Close()

	metaRepo := postgres.NewContentRepository(pgPool)
	catalogRepo := infraMongo.NewCatalogRepository(mongoDB)
	cacheRepo := infraRedis.NewCacheRepository(redisClient)
	catalogUC := usecase.NewCatalogUseCase(catalogRepo, cacheRepo)

	shortIDs, err := metaRepo.ListAllShortIDs(context.Background())
	if err != nil {
		log.Fatalf("failed to list short IDs: %v", err)
	}

	log.Printf("Starting batch sync for %d items...", len(shortIDs))

	const numWorkers = 5
	jobs := make(chan string, len(shortIDs))
	var wg sync.WaitGroup

	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for id := range jobs {
				jobCtx, jobCancel := context.WithTimeout(context.Background(), 10*time.Second)
				if err := catalogUC.SyncContent(jobCtx, id, metaRepo); err != nil {
					log.Printf("[Worker %d] ERROR syncing %s: %v", workerID, id, err)
				} else {
					log.Printf("[Worker %d] SUCCESS synchronized %s", workerID, id)
				}
				jobCancel()
			}
		}(w)
	}

	for _, id := range shortIDs {
		jobs <- id
	}
	close(jobs)
	wg.Wait()

	log.Println("Batch sync completed successfully.")
}
