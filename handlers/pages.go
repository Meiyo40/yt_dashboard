package handlers

import (
	"log"
	"net/http"

	"yt_dashboard/db"
	"yt_dashboard/templates/pages"
	"yt_dashboard/youtube"
)

// IndexPageHandler handles requests to the root URL (GET /)
func IndexPageHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Query total tracked comments
	var commentsCount int
	err := db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM own_comments").Scan(&commentsCount)
	if err != nil {
		log.Printf("Error querying own_comments count: %v", err)
		commentsCount = 0
	}

	// Query total unseen replies
	var repliesCount int
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM replies WHERE seen = 0").Scan(&repliesCount)
	if err != nil {
		log.Printf("Error querying unseen replies count: %v", err)
		repliesCount = 0
	}

	// Check if OAuth tokens exist (to determine connection status)
	var tokensCount int
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM oauth_tokens").Scan(&tokensCount)
	if err != nil {
		log.Printf("Error querying oauth_tokens count: %v", err)
		tokensCount = 0
	}
	isConnected := tokensCount > 0

	// Live quota usage from in-memory counter
	quotaUsage := youtube.QuotaCounter

	// Render the page template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = pages.Index(commentsCount, repliesCount, quotaUsage, isConnected).Render(ctx, w)
	if err != nil {
		log.Printf("Error rendering index template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
