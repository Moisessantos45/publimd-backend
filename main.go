package main

import (
	"context"
	"publimd/config"
	"publimd/config/db"
	"publimd/internal/features/post"
	"publimd/internal/routes"
	"publimd/internal/shared/middleware"
	"publimd/internal/shared/socket"
	"publimd/internal/shared/utils"

	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Advertencia: no se encontró .env, se usan variables de entorno del sistema")
	}

	if err := db.Connect(); err != nil {
		fmt.Println("Error initializing database:", err)
		return
	}

	if err := db.InitializeDatabase(); err != nil {
		fmt.Println("Error initializing database:", err)
		return
	}

	if err := config.InitRedis(context.Background()); err != nil {
		fmt.Println("Error initializing Redis:", err)
		return
	}

	// if err := utils.GeneratePDFFromMarkdownContent("# Hola\n**bold**", "test.pdf"); err != nil {
	// 	fmt.Println("Error generating PDF:", err)
	// 	return
	// }

	// embeddings.TestGeneratePostEmbedding()

	utils.InitMailer(3)

	HOST_URL_DEV := os.Getenv("HOST_URL_DEV")
	HOST_URL_PROD := os.Getenv("HOST_URL_PROD")
	HOST_URL_PROD_WWW := os.Getenv("HOST_URL_PROD_WWW")

	// log.Printf("HOST_URL_DEV: %s", HOST_URL_DEV)
	log.Printf("HOST_URL_PROD: %s", HOST_URL_PROD)
	log.Printf("HOST_URL_PROD_WWW: %s", HOST_URL_PROD_WWW)

	allowedOrigins := []string{HOST_URL_PROD, HOST_URL_PROD_WWW, HOST_URL_DEV}
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(gzip.Gzip(gzip.DefaultCompression))

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	r.Use(middleware.RateLimiterMiddleware(rate.Every(time.Second/2), 30))

	// iniciliza las dependencias y casos de uso
	routes.Init()
	api := r.Group("/api/v1")
	{
		routes.UserRoutes(api)
		routes.PostRoutes(api)
		routes.CollaboratorRoutes(api)
	}

	auth := r.Group("/api/v1/auth")
	{
		routes.AuthRoutes(auth)
	}
	wsHub := socket.NewHub()
	go wsHub.Run()

	postSvc, permChecker := routes.GetSocketDeps()
	wsHandler := socket.NewHandler(
		wsHub,
		allowedOrigins,
		postSvc,
		permChecker,
	)

	ws := r.Group("/api/v1/ws")
	{
		ws.GET("/editor", func(c *gin.Context) {
			wsHandler.HandleEditor(c.Writer, c.Request)
		})
	}

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Welcome to the SpinLuck API!",
		})
	})

	var wg sync.WaitGroup
	wg.Go(func() {
		middleware.StartCleanup()
	})

	workerCtx, workerCancel := context.WithCancel(context.Background())
	pRepo, embCl := routes.GetWorkerDeps()
	w := post.NewOutboxWorker(pRepo, embCl)
	wg.Go(func() {
		if err := w.Run(workerCtx); err != nil && err != context.Canceled {
			log.Printf("outbox worker exited with error: %v", err)
		}
	})

	log.Println("Server starting on :4104...")
	srv := &http.Server{
		Addr:    ":4104",
		Handler: r,
		// ReadTimeout: 10 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed: ", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")
	workerCancel()

	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	wg.Wait()
	log.Println("Server exiting")
}
