package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "path/filepath"
    "strings"
    "syscall"
    "time"

    "github.com/Cdaprod/registry-service/internal/api"
    "github.com/Cdaprod/registry-service/internal/storage"
    "github.com/Cdaprod/registry-service/pkg/builtins"
    "github.com/Cdaprod/registry-service/pkg/logger"
    "github.com/gorilla/mux"
    "github.com/rs/cors"
    "go.uber.org/zap"
)

func setCorrectMIMEType(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Set correct MIME type
        switch {
        case strings.HasSuffix(r.URL.Path, ".js"):
            w.Header().Set("Content-Type", "application/javascript")
        case strings.HasSuffix(r.URL.Path, ".css"):
            w.Header().Set("Content-Type", "text/css")
        case strings.HasSuffix(r.URL.Path, ".html"):
            w.Header().Set("Content-Type", "text/html")
        case strings.HasSuffix(r.URL.Path, ".json"):
            w.Header().Set("Content-Type", "application/json")
        case strings.HasSuffix(r.URL.Path, ".png"):
            w.Header().Set("Content-Type", "image/png")
        case strings.HasSuffix(r.URL.Path, ".jpg"), strings.HasSuffix(r.URL.Path, ".jpeg"):
            w.Header().Set("Content-Type", "image/jpeg")
        case strings.HasSuffix(r.URL.Path, ".gif"):
            w.Header().Set("Content-Type", "image/gif")
        case strings.HasSuffix(r.URL.Path, ".svg"):
            w.Header().Set("Content-Type", "image/svg+xml")
        default:
            w.Header().Set("Content-Type", "text/plain") // Default MIME type
        }
        next.ServeHTTP(w, r)
    })
}

// initializeServer sets up and starts the HTTP server with all configurations.
func initializeServer(router http.Handler, bindAddr string, l *zap.Logger) *http.Server {
    l.Info("Starting server", zap.String("bind_address", bindAddr))

    server := &http.Server{
        Addr:    bindAddr,
        Handler: router,
    }

    go func() {
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            l.Fatal("Failed to start server", zap.Error(err))
        }
    }()

    l.Info("Server is ready to handle requests")
    return server
}

// handleGracefulShutdown gracefully shuts down the server on receiving a termination signal.
func handleGracefulShutdown(server *http.Server, l *zap.Logger) {
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    <-quit

    l.Info("Server is shutting down...")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        l.Fatal("Server forced to shutdown", zap.Error(err))
    }

    l.Info("Server has shut down gracefully")
}

// main is the entry point for the application.
func main() {
    // Initialize logger
    l, err := logger.NewLogger()
    if err != nil {
        log.Fatalf("Failed to initialize logger: %v", err)
    }
    defer l.Sync()

    // Initialize in-memory storage
    memoryStorage := storage.NewMemoryStorage()

    // Load built-in plugins
    builtinLoader := builtins.NewBuiltinLoader(memoryStorage, "pkg/plugins/")
    if err := builtinLoader.LoadAll(); err != nil {
        l.Fatal("Error loading built-ins", zap.Error(err))
    }

    // Set up router using mux
    r := mux.NewRouter()
    api.SetupRoutes(r, memoryStorage, l)

    // Serve static files from the web/build directory with correct MIME types
    fs := http.FileServer(http.Dir("./web/build"))
    r.PathPrefix("/static/").Handler(setCorrectMIMEType(http.StripPrefix("/static", fs)))

    // Serve index.html for any non-static file requests (React Router fallback)
    r.PathPrefix("/").Handler(setCorrectMIMEType(http.StripPrefix("/", http.FileServer(http.Dir("./web/build")))))

    // Determine port and bind address
    port := os.Getenv("PORT")
    if port == "" {
        port = "7777"
    }
    bindAddr := "0.0.0.0:" + port

    // Set up CORS
    c := cors.New(cors.Options{
        AllowedOrigins:   []string{"*"},  // Allow all origins for staged environment
        AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{"Content-Type", "Authorization"},
        AllowCredentials: true,
    })

    // Wrap router with CORS handler
    handler := c.Handler(r)

    // Start the HTTP server
    server := initializeServer(handler, bindAddr, l)

    // Handle graceful shutdown
    handleGracefulShutdown(server, l)
}