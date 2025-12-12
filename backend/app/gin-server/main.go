package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/yoockh/yoospeak/config"
	"github.com/yoockh/yoospeak/internal/api/handlers"
	"github.com/yoockh/yoospeak/internal/api/middleware"
	"github.com/yoockh/yoospeak/internal/api/routes"
	"github.com/yoockh/yoospeak/internal/cache"
	"github.com/yoockh/yoospeak/internal/logger"
	mongorepo "github.com/yoockh/yoospeak/internal/repositories/mongo"
	pgrepo "github.com/yoockh/yoospeak/internal/repositories/postgres"
	"github.com/yoockh/yoospeak/internal/services"

	llmprov "github.com/yoockh/yoospeak/internal/providers/llm"
	sttprov "github.com/yoockh/yoospeak/internal/providers/stt"
	storagepkg "github.com/yoockh/yoospeak/internal/storage"
	"github.com/yoockh/yoospeak/internal/workers"
)

func main() {
	_ = godotenv.Load()

	l := logger.New()

	// Init MongoDB
	if err := config.InitMongo(); err != nil {
		log.Fatalf("MongoDB init error: %v", err)
	}
	// Init PostgreSQL
	if err := config.InitPostgres(); err != nil {
		log.Fatalf("PostgreSQL init error: %v", err)
	}
	// Init Redis
	if err := config.InitRedis(); err != nil {
		log.Fatalf("Redis init error: %v", err)
	}

	// Ensure Mongo indexes (TTL)
	if err := config.EnsureMongoIndexes(); err != nil {
		log.Fatalf("Mongo indexes error: %v", err)
	}

	// Mongo DB
	dbName := os.Getenv("MONGO_URI")
	if dbName == "" {
		dbName = "yoospeak"
	}
	mdb := config.MongoClient.Database(dbName)

	// GCS uploader (required for CV upload)
	bucket := os.Getenv("GCS_BUCKET")
	if bucket == "" {
		log.Fatalf("GCS_BUCKET is required (for CV upload)")
	}
	ctx := context.Background()
	gcsUp, err := storagepkg.NewGCSUploader(ctx, bucket)
	if err != nil {
		log.Fatalf("GCS uploader init error: %v", err)
	}
	defer gcsUp.Close()

	// Repos
	sessionRepo := mongorepo.NewSessionRepo(mdb)
	bufferRepo := mongorepo.NewBufferRepo(mdb)

	profileRepo := pgrepo.NewProfileRepo(config.PostgresDB)
	convoRepo := pgrepo.NewConversationRepo(config.PostgresDB)
	cvRepo := pgrepo.NewCVFileRepo(config.PostgresDB)

	// Cache (optional)
	redisCache := cache.NewRedisCache(config.RedisClient)

	// Services
	sessionSvc := services.NewSessionService(sessionRepo)
	bufferSvc := services.NewBufferService(bufferRepo, 24*time.Hour)
	profileSvc := services.NewProfileServiceWithCache(profileRepo, redisCache, 5*time.Minute)
	convoSvc := services.NewConversationService(convoRepo)
	cvSvc := services.NewCVFileService(cvRepo, gcsUp)

	// Handlers
	sessionH := handlers.NewSessionHandler(sessionSvc)
	profileH := handlers.NewProfileHandler(profileSvc)
	convoH := handlers.NewConversationHandler(convoSvc)
	wsH := handlers.NewWSHandler(sessionSvc, bufferSvc, config.RedisClient)
	cvH := handlers.NewCVHandler(cvSvc)

	// Gin
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger(l))

	routes.RegisterRoutes(r, routes.Deps{
		Session:      sessionH,
		Profile:      profileH,
		Conversation: convoH,
		WS:           wsH,
		CV:           cvH,
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Optional: start workers in same process
	var sttP sttprov.Provider
	var llmP llmprov.Provider
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if os.Getenv("RUN_WORKERS") == "1" {
		var err error

		sttP, err = sttprov.NewGoogleSpeech(ctx)
		if err != nil {
			log.Fatalf("STT init error: %v", err)
		}

		projectID := os.Getenv("VERTEX_PROJECT_ID")
		location := os.Getenv("VERTEX_LOCATION")
		model := os.Getenv("VERTEX_GEMINI_MODEL")
		if projectID == "" || location == "" {
			log.Fatalf("VERTEX_PROJECT_ID and VERTEX_LOCATION are required when RUN_WORKERS=1")
		}

		llmP, err = llmprov.NewVertexGemini(ctx, projectID, location, model)
		if err != nil {
			log.Fatalf("LLM init error: %v", err)
		}

		pool := &workers.AudioWorkerPool{
			Redis:      config.RedisClient,
			Buffers:    bufferSvc,
			NumWorkers: 5,
			STT:        sttP,
			LLM:        llmP,
			Logger:     l,
			Stream:     "audio:stream",
			Group:      "audio-workers",
		}
		if err := pool.Start(ctx); err != nil {
			log.Fatalf("Workers start error: %v", err)
		}
	}

	// Serve + graceful shutdown
	go func() {
		l.WithField("addr", srv.Addr).Info("server started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	_ = srv.Shutdown(shutdownCtx)

	if llmP != nil {
		_ = llmP.Close()
	}
	if sttP != nil {
		_ = sttP.Close()
	}
	if config.RedisClient != nil {
		_ = config.RedisClient.Close()
	}
	if config.MongoClient != nil {
		_ = config.MongoClient.Disconnect(shutdownCtx)
	}
}
