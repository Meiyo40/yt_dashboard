package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"yt_dashboard/db"
	"yt_dashboard/handlers"
)

func main() {
	// 1. Load environment variables from .env
	loadEnv(".env")

	// Get configuration variables
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "yt_dashboard.db"
	}

	// 2. Initialize Database
	database, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer database.Close()

	// 3. Set up Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Web routes
	r.Get("/", handlers.IndexPageHandler)

	// Comment registry
	r.Get("/comments", handlers.CommentsPageHandler)

	// HTMX Partials
	r.Get("/partials/feed", handlers.FeedPartialHandler)
	r.Get("/partials/feed/sync", handlers.FeedSyncHandler)
	r.Post("/partials/replies/{id}/seen", handlers.MarkReplySeenHandler)
	r.Post("/partials/comments/add", handlers.CommentsAddHandler)
	r.Delete("/partials/comments/{id}", handlers.CommentsDeleteHandler)

	// OAuth login/callback/logout routes
	r.Get("/oauth2/login", handlers.OAuthLoginHandler)
	r.Get("/oauth2/callback", handlers.OAuthCallbackHandler)
	r.Get("/oauth2/logout", handlers.OAuthLogoutHandler)

	// Static files file server
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "static"))
	fileServer(r, "/static", filesDir)

	// 4. Start Server
	serverAddr := ":" + port
	log.Printf("Starting server on http://localhost%s", serverAddr)
	if err := http.ListenAndServe(serverAddr, r); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server failed: %v", err)
	}
}

// loadEnv parses a local key=value env file and loads it into environment variables.
func loadEnv(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		// If the file doesn't exist, we skip silently and rely on system env vars or defaults
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comment lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// Remove wrapping quotes if present
		val = strings.Trim(val, `"'`)
		os.Setenv(key, val)
	}
}

// fileServer sets up a http.FileServer handler for a route and strips prefix
func fileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("fileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
func contextWithCancel(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}
func fmtErrorf(format string, a ...interface{}) error {
	return fmt.Errorf(format, a...)
}
