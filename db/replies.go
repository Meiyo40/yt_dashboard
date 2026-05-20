package db

import (
	"context"
	"time"
)

func SaveReplies(ctx context.Context, replies []Reply) (int, error) {
	if len(replies) == 0 {
		return 0, nil
	}

	query := `INSERT OR IGNORE INTO replies (id, parent_id, author_display_name, author_profile_image_url, author_channel_id, text_display, published_at, seen) VALUES (?, ?, ?, ?, ?, ?, ?, 0)`

	count := 0
	for _, r := range replies {
		publishedAtStr := r.PublishedAt.Format(time.RFC3339)
		result, err := DB.ExecContext(ctx, query, r.ID, r.ParentID, r.AuthorDisplayName, r.AuthorProfileImageURL, r.AuthorChannelID, r.TextDisplay, publishedAtStr)
		if err != nil {
			return count, err
		}
		n, _ := result.RowsAffected()
		if n > 0 {
			count++
		}
	}
	return count, nil
}

func MarkReplySeen(ctx context.Context, replyID string) error {
	_, err := DB.ExecContext(ctx, "UPDATE replies SET seen = 1 WHERE id = ?", replyID)
	return err
}

func GetUnseenReplyCount(ctx context.Context) (int, error) {
	var count int
	err := DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM replies WHERE seen = 0").Scan(&count)
	return count, err
}
