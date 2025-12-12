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
	"go.mongodb.org/mongo-driver/mongo"

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
		l.WithError(err).Error("MongoDB init failed")
		// Don't fatal - let app start for health checks
	}
	// Init PostgreSQL
	if err := config.InitPostgres(); err != nil {
		l.WithError(err).Error("PostgreSQL init failed")
	}
	// Init Redis
	if err := config.InitRedis(); err != nil {
		l.WithError(err).Error("Redis init failed")
	}

	// Ensure Mongo indexes (TTL) - only if MongoDB is available
	var mdb *mongo.Database
	if config.MongoClient != nil {
		if err := config.EnsureMongoIndexes(); err != nil {
			l.WithError(err).Error("Mongo indexes error")
		}
		dbName := os.Getenv("MONGO_DB")
		if dbName == "" {
			dbName = "yoospeak"
		}
		l.WithField("mongo_db", dbName).Info("Using MongoDB database")
		mdb = config.MongoClient.Database(dbName)
	} else {
		l.Warn("MongoDB not available - some features will be disabled")
	}

	// GCS uploader (optional for CV upload)
	var gcsUp *storagepkg.GCSUploader
	bucket := os.Getenv("GCS_BUCKET")
	if bucket != "" {
		baseCtx := context.Background()
		var err error
		gcsUp, err = storagepkg.NewGCSUploader(baseCtx, bucket)
		if err != nil {
			l.WithError(err).Error("GCS uploader init failed")
		}
		if gcsUp != nil {
			defer gcsUp.Close()
		}
	}

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
			l.WithError(err).Error("STT init failed")
		} else {
			projectID := os.Getenv("VERTEX_PROJECT_ID")
			location := os.Getenv("VERTEX_LOCATION")
			model := os.Getenv("VERTEX_GEMINI_MODEL")
			if projectID != "" && location != "" {
				llmP, err = llmprov.NewVertexGemini(ctx, projectID, location, model)
				if err != nil {
					l.WithError(err).Error("LLM init failed")
				} else if config.RedisClient != nil {
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
						l.WithError(err).Error("Workers start failed")
					}
				}
			}
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
