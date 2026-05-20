package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"yt_dashboard/db"
	"yt_dashboard/templates/partials"
	"yt_dashboard/youtube"
)

// FeedPartialHandler handles requests to GET /partials/feed (fast DB-only, no API calls)
func FeedPartialHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Query replies joined with parent own_comments
	rows, err := db.DB.QueryContext(ctx, `
		SELECT r.id, r.parent_id, r.author_display_name, r.author_profile_image_url, r.author_channel_id, 
		       r.text_display, r.published_at, r.seen,
		       c.comment_id, c.video_id, c.video_title, c.channel_title, c.text_original, c.posted_at, c.last_checked_at, c.reply_count
		FROM replies r
		JOIN own_comments c ON r.parent_id = c.comment_id
		ORDER BY r.published_at DESC
		LIMIT 50
	`)
	if err != nil {
		log.Printf("Error querying notification feed: %v", err)
		http.Error(w, "Failed to load feed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	notifications := scanNotifications(rows)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = partials.Feed(notifications, "").Render(ctx, w)
	if err != nil {
		log.Printf("Error rendering feed partial: %v", err)
		http.Error(w, "Failed to render feed", http.StatusInternalServerError)
	}
}

// FeedSyncHandler handles requests to GET /partials/feed/sync
// It polls YouTube API for new replies, saves them, then renders the feed.
func FeedSyncHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	token, err := db.GetValidToken(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			renderFeedWithAlert(ctx, w, "YouTube account not connected. Please connect your account to sync replies.")
		} else {
			log.Printf("Error getting valid token for sync: %v", err)
			renderFeedWithAlert(ctx, w, "Authentication error. Please reconnect your YouTube account.")
		}
		return
	}

	newCount, err := runFeedSync(ctx, token.AccessToken)
	if err != nil {
		log.Printf("Feed sync error: %v", err)
		renderFeedWithAlert(ctx, w, "Failed to sync with YouTube API. Some replies may be stale.")
		return
	}

	// Query the updated feed
	rows, err := db.DB.QueryContext(ctx, `
		SELECT r.id, r.parent_id, r.author_display_name, r.author_profile_image_url, r.author_channel_id, 
		       r.text_display, r.published_at, r.seen,
		       c.comment_id, c.video_id, c.video_title, c.channel_title, c.text_original, c.posted_at, c.last_checked_at, c.reply_count
		FROM replies r
		JOIN own_comments c ON r.parent_id = c.comment_id
		ORDER BY r.published_at DESC
		LIMIT 50
	`)
	if err != nil {
		log.Printf("Error querying notification feed after sync: %v", err)
		renderFeedWithAlert(ctx, w, "Sync succeeded but failed to load feed.")
		return
	}
	defer rows.Close()

	notifications := scanNotifications(rows)

	var alert string
	if newCount > 0 {
		alert = fmt.Sprintf("Synced %d new reply(s).", newCount)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = partials.Feed(notifications, alert).Render(ctx, w)
	if err != nil {
		log.Printf("Error rendering feed partial after sync: %v", err)
		http.Error(w, "Failed to render feed", http.StatusInternalServerError)
	}
}

// MarkReplySeenHandler handles POST /partials/replies/{id}/seen
func MarkReplySeenHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	replyID := chi.URLParam(r, "id")
	if replyID == "" {
		http.Error(w, "Reply ID is required", http.StatusBadRequest)
		return
	}

	if err := db.MarkReplySeen(ctx, replyID); err != nil {
		log.Printf("Error marking reply %s as seen: %v", replyID, err)
		http.Error(w, "Failed to mark reply as seen", http.StatusInternalServerError)
		return
	}

	// Return empty 200 - HTMX will use hx-swap to update the UI
	w.WriteHeader(http.StatusOK)
}

// runFeedSync iterates over all tracked comments, fetches replies from YouTube,
// saves new ones to the DB, and updates last_checked_at.
// Returns the number of newly saved replies.
func runFeedSync(ctx context.Context, accessToken string) (int, error) {
	comments, err := db.GetComments(ctx)
	if err != nil {
		return 0, err
	}

	totalNew := 0
	for _, comment := range comments {
		replies, err := youtube.FetchReplies(ctx, accessToken, comment.CommentID)
		if err != nil {
			// Log per-comment errors but continue syncing the rest
			log.Printf("Error fetching replies for comment %s: %v", comment.CommentID, err)
			continue
		}

		if len(replies) > 0 {
			newCount, err := db.SaveReplies(ctx, replies)
			if err != nil {
				log.Printf("Error saving replies for comment %s: %v", comment.CommentID, err)
				continue
			}
			totalNew += newCount
		}

		// Update last_checked_at regardless of success/failure of save
		if err := db.UpdateCheckedAt(ctx, comment.CommentID); err != nil {
			log.Printf("Error updating checked_at for comment %s: %v", comment.CommentID, err)
		}
	}

	return totalNew, nil
}

// scanNotifications reads notification rows from a sql.Rows and returns a slice.
func scanNotifications(rows *sql.Rows) []db.NotificationItem {
	var notifications []db.NotificationItem
	for rows.Next() {
		var item db.NotificationItem
		var seenInt int
		var publishedAtStr, postedAtStr string
		var lastCheckedAtStr sql.NullString
		var videoTitleStr, channelTitleStr sql.NullString

		err := rows.Scan(
			&item.Reply.ID, &item.Reply.ParentID, &item.Reply.AuthorDisplayName, &item.Reply.AuthorProfileImageURL, &item.Reply.AuthorChannelID,
			&item.Reply.TextDisplay, &publishedAtStr, &seenInt,
			&item.OwnComment.CommentID, &item.OwnComment.VideoID, &videoTitleStr, &channelTitleStr, &item.OwnComment.TextOriginal, &postedAtStr, &lastCheckedAtStr, &item.OwnComment.ReplyCount,
		)
		if err != nil {
			log.Printf("Error scanning notification row: %v", err)
			continue
		}

		// Convert seen integer to boolean
		item.Reply.Seen = seenInt == 1
		item.Seen = item.Reply.Seen

		// Parse times
		if t, err := time.Parse(time.RFC3339, publishedAtStr); err == nil {
			item.Reply.PublishedAt = t
		}
		if t, err := time.Parse(time.RFC3339, postedAtStr); err == nil {
			item.OwnComment.PostedAt = t
		}
		if lastCheckedAtStr.Valid {
			if t, err := time.Parse(time.RFC3339, lastCheckedAtStr.String); err == nil {
				item.OwnComment.LastCheckedAt = &t
			}
		}

		// Handle Null strings
		if videoTitleStr.Valid {
			item.OwnComment.VideoTitle = videoTitleStr.String
		}
		if channelTitleStr.Valid {
			item.OwnComment.ChannelTitle = channelTitleStr.String
		}

		notifications = append(notifications, item)
	}
	return notifications
}

// renderFeedWithAlert renders the current DB feed with an alert banner.
func renderFeedWithAlert(ctx context.Context, w http.ResponseWriter, alert string) {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT r.id, r.parent_id, r.author_display_name, r.author_profile_image_url, r.author_channel_id, 
		       r.text_display, r.published_at, r.seen,
		       c.comment_id, c.video_id, c.video_title, c.channel_title, c.text_original, c.posted_at, c.last_checked_at, c.reply_count
		FROM replies r
		JOIN own_comments c ON r.parent_id = c.comment_id
		ORDER BY r.published_at DESC
		LIMIT 50
	`)
	if err != nil {
		log.Printf("Error querying feed for alert render: %v", err)
		http.Error(w, "Failed to load feed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	notifications := scanNotifications(rows)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = partials.Feed(notifications, alert).Render(ctx, w)
}
