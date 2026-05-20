package handlers

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"yt_dashboard/db"
	"yt_dashboard/templates/pages"
	"yt_dashboard/templates/partials"
	"yt_dashboard/youtube"
)

func CommentsPageHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	comments, err := db.GetComments(ctx)
	if err != nil {
		log.Printf("Error querying comments: %v", err)
		comments = []db.OwnComment{}
	}

	var tokensCount int
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM oauth_tokens").Scan(&tokensCount)
	if err != nil {
		tokensCount = 0
	}
	isConnected := tokensCount > 0

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = pages.Comments(comments, isConnected).Render(ctx, w)
	if err != nil {
		log.Printf("Error rendering comments template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func CommentsAddHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	rawURL := r.FormValue("url")
	if rawURL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	videoID, commentID, ok := youtube.ParseCommentURL(rawURL)
	if !ok {
		renderCommentsOrError(w, r, "Could not parse a YouTube comment URL. Expected format: https://www.youtube.com/watch?v=VIDEO_ID&lc=COMMENT_ID")
		return
	}

	token, err := db.GetValidToken(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			renderCommentsOrError(w, r, "YouTube account not connected. Please log in first.")
		} else {
			log.Printf("Error getting valid token: %v", err)
			renderCommentsOrError(w, r, "Authentication error. Please try reconnecting your YouTube account.")
		}
		return
	}

	commentSnippet, err := youtube.FetchCommentSnippet(ctx, token.AccessToken, commentID)
	if err != nil {
		log.Printf("Error fetching comment %s: %v", commentID, err)
		renderCommentsOrError(w, r, "Could not fetch the comment. Check the URL or ensure the comment is public.")
		return
	}

	publishedAt, err := time.Parse(time.RFC3339, commentSnippet.PublishedAt)
	if err != nil {
		log.Printf("Error parsing publishedAt %s: %v", commentSnippet.PublishedAt, err)
		publishedAt = time.Now()
	}

	comment := &db.OwnComment{
		CommentID:    commentID,
		VideoID:      videoID,
		TextOriginal: commentSnippet.TextOriginal,
		PostedAt:     publishedAt,
		ReplyCount:   0,
	}

	videoSnippet, err := youtube.FetchVideoSnippet(ctx, token.AccessToken, videoID)
	if err != nil {
		log.Printf("Error fetching video %s: %v (comment will be stored without video title)", videoID, err)
	} else {
		comment.VideoTitle = videoSnippet.Title
	}

	if err := db.SaveComment(ctx, comment); err != nil {
		log.Printf("Error saving comment: %v", err)
		renderCommentsOrError(w, r, "Database error. Failed to save the comment.")
		return
	}

	renderCommentList(w, r)
}

func CommentsDeleteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	commentID := chi.URLParam(r, "id")
	if commentID == "" {
		http.Error(w, "Comment ID is required", http.StatusBadRequest)
		return
	}

	if err := db.DeleteComment(ctx, commentID); err != nil {
		log.Printf("Error deleting comment %s: %v", commentID, err)
		http.Error(w, "Failed to delete comment", http.StatusInternalServerError)
		return
	}

	renderCommentList(w, r)
}

func renderCommentList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	comments, err := db.GetComments(ctx)
	if err != nil {
		log.Printf("Error querying comments: %v", err)
		comments = []db.OwnComment{}
	}

	var tokensCount int
	_ = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM oauth_tokens").Scan(&tokensCount)
	isConnected := tokensCount > 0

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = partials.CommentList(comments, isConnected, "").Render(ctx, w)
	if err != nil {
		log.Printf("Error rendering comment list: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func renderCommentsOrError(w http.ResponseWriter, r *http.Request, errorMsg string) {
	ctx := r.Context()

	comments, err := db.GetComments(ctx)
	if err != nil {
		comments = []db.OwnComment{}
	}

	var tokensCount int
	_ = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM oauth_tokens").Scan(&tokensCount)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)

	partials.CommentList(comments, tokensCount > 0, errorMsg).Render(ctx, w)
}
