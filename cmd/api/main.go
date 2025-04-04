// cmd/api/main.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/parvez-capri/ronnin/internal/config"
	"github.com/parvez-capri/ronnin/internal/handlers"
	"github.com/parvez-capri/ronnin/internal/services"
	"github.com/parvez-capri/ronnin/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	_ "github.com/parvez-capri/ronnin/docs"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)
)

// @title           Ronnin API
// @version         1.0
// @description     API Server for issue reporting with Jira integration, MongoDB persistence, and S3 file uploads
// @termsOfService  http://swagger.io/terms/

// @contact.name   Your Organization Name
// @contact.url    http://www.yourorg.com/support
// @contact.email  support@yourorg.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /

// @tag.name        tickets
// @tag.description Ticket viewing endpoints - for accessing stored reports

// @tag.name        reports
// @tag.description Issue reporting with file uploads

// @tag.name        health
// @tag.description Health check and monitoring endpoints

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

// @x-extension-openapi {"example": "value on a json format"}

func main() {
	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Failed to load configuration:", err)
		os.Exit(1)
	}

	// Initialize logger
	log, err := logger.NewLogger(cfg.LogLevel, cfg.Environment)
	if err != nil {
		fmt.Println("Failed to initialize logger:", err)
		os.Exit(1)
	}
	defer log.Sync()

	// Set Gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	r := gin.New()

	// Middleware
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Initialize validator
	validate := validator.New()

	// Initialize MongoDB service if configured
	var mongoService *services.MongoDBService
	if cfg.MongoURI != "" {
		log.Info("Initializing MongoDB service",
			zap.String("uri", cfg.MongoURI),
			zap.String("database", cfg.MongoDB),
			zap.String("collection", cfg.MongoCollection))

		mongoService, err = services.NewMongoDBService(
			cfg.MongoURI,
			cfg.MongoDB,
			cfg.MongoCollection,
		)
		if err != nil {
			log.Warn("Failed to initialize MongoDB service, database persistence will be disabled", zap.Error(err))
		} else {
			log.Info("MongoDB service initialized successfully")

			// Test connection
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			tickets, err := mongoService.GetAllTickets(ctx)
			if err != nil {
				log.Warn("Failed to retrieve tickets from MongoDB", zap.Error(err))
			} else {
				log.Info("Successfully connected to MongoDB", zap.Int("ticket_count", len(tickets)))
			}
		}
	} else {
		log.Warn("MongoDB configuration not provided, database persistence will be disabled")
	}

	// Initialize Jira service
	jiraService, err := services.NewJiraService(
		cfg.JiraURL,
		cfg.JiraUsername,
		cfg.JiraAPIToken,
		cfg.JiraProjectKey,
		cfg.SupportTeamMembers,
		cfg.DefaultPriority,
		mongoService,
	)
	if err != nil {
		log.Fatal("Failed to initialize Jira service", zap.Error(err))
	}

	// Initialize S3 service if configured
	var s3Service *services.S3Service
	if cfg.AWSS3AccessKey != "" && cfg.AWSS3SecretKey != "" {
		s3Service, err = services.NewS3Service(
			cfg.AWSS3AccessKey,
			cfg.AWSS3SecretKey,
			cfg.AWSS3Region,
			cfg.AWSS3BucketName,
			cfg.AWSS3BaseURL,
		)
		if err != nil {
			log.Warn("Failed to initialize S3 service, file uploads will be disabled", zap.Error(err))
		} else {
			log.Info("S3 service initialized successfully",
				zap.String("region", cfg.AWSS3Region),
				zap.String("bucket", cfg.AWSS3BucketName),
			)
		}
	} else {
		log.Warn("S3 configuration not provided, file uploads will be disabled")
	}

	// Initialize handlers
	ticketHandler := handlers.NewTicketHandler(jiraService, log, validate)
	reportHandler := handlers.NewReportHandler(jiraService, s3Service, log, validate)

	// Routes
	r.GET("/health", handlers.HealthCheckGin)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.POST("/report-issue", reportHandler.ReportIssue)

	// MongoDB routes
	r.GET("/tickets", ticketHandler.GetAllTicketsGin)
	r.GET("/tickets/:id", ticketHandler.GetTicketByIDGin)

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// HTTP Server configuration
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Info("Starting server", zap.Int("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Shutdown server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server shutdown failed", zap.Error(err))
	}

	if err := jiraService.Cleanup(); err != nil {
		log.Error("Failed to cleanup Jira service", zap.Error(err))
	}

	// Cleanup MongoDB connection if initialized
	if mongoService != nil {
		if err := mongoService.Disconnect(context.Background()); err != nil {
			log.Error("Failed to disconnect from MongoDB", zap.Error(err))
		} else {
			log.Info("MongoDB connection closed")
		}
	}

	log.Info("Server stopped gracefully")
}
