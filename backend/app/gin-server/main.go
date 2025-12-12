package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/yoockh/yoospeak/backend/config"
)

func main() {
	_ = godotenv.Load()

	// Init MongoDB
	if err := config.InitMongo(); err != nil {
		log.Fatalf("MongoDB init error: %v", err)
	}
	fmt.Println("MongoDB connected")

	// Init PostgreSQL
	if err := config.InitPostgres(); err != nil {
		log.Fatalf("PostgreSQL init error: %v", err)
	}
	fmt.Println("PostgreSQL connected")

	// Init Redis
	if err := config.InitRedis(); err != nil {
		log.Fatalf("Redis init error: %v", err)
	}
	fmt.Println("Redis connected")

	// Start Gin server
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
