package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"yt_dashboard/db"
	"yt_dashboard/templates/partials"
)

// FeedPartialHandler handles requests to GET /partials/feed
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = partials.Feed(notifications).Render(ctx, w)
	if err != nil {
		log.Printf("Error rendering feed partial: %v", err)
		http.Error(w, "Failed to render feed", http.StatusInternalServerError)
	}
}
