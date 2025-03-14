package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/blockdaemon/s3-bucket-browser/internal/api"
	"github.com/blockdaemon/s3-bucket-browser/internal/cache"
	"github.com/blockdaemon/s3-bucket-browser/internal/config"
	"github.com/blockdaemon/s3-bucket-browser/internal/s3"
	"github.com/gorilla/mux"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.json", "Path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create S3 service
	s3Service, err := s3.NewService(&cfg.S3)
	if err != nil {
		log.Fatalf("Failed to create S3 service: %v", err)
	}

	// Create Redis cache (optional)
	var cacheService *cache.RedisCache
	cacheService, err = cache.NewRedisCache(&cfg.Redis)
	if err != nil {
		log.Printf("Warning: Failed to create Redis cache: %v", err)
		log.Println("Continuing without Redis cache")
		cacheService = nil
	} else {
		defer cacheService.Close()
	}

	// Create API handler
	handler := api.NewHandler(s3Service, cacheService)

	// Create router
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Serve static files for the frontend
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("../frontend/dist")))

	// Create server
	server := &http.Server{
		Addr:         ":" + strconv.Itoa(cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on port %d", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
