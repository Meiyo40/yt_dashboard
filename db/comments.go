package db

import (
	"context"
	"time"
)

func SaveComment(ctx context.Context, comment *OwnComment) error {
	query := `
		INSERT INTO own_comments (comment_id, video_id, video_title, channel_title, text_original, posted_at, last_checked_at, reply_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(comment_id) DO UPDATE SET
			video_id = excluded.video_id,
			video_title = CASE WHEN excluded.video_title <> '' THEN excluded.video_title ELSE video_title END,
			channel_title = CASE WHEN excluded.channel_title <> '' THEN excluded.channel_title ELSE channel_title END,
			text_original = excluded.text_original,
			posted_at = excluded.posted_at,
			reply_count = excluded.reply_count
	`
	postedAtStr := comment.PostedAt.Format(time.RFC3339)
	var lastCheckedAtStr interface{}
	if comment.LastCheckedAt != nil {
		lastCheckedAtStr = comment.LastCheckedAt.Format(time.RFC3339)
	}

	_, err := DB.ExecContext(ctx, query,
		comment.CommentID, comment.VideoID, comment.VideoTitle, comment.ChannelTitle,
		comment.TextOriginal, postedAtStr, lastCheckedAtStr, comment.ReplyCount,
	)
	return err
}

func GetComments(ctx context.Context) ([]OwnComment, error) {
	query := `SELECT comment_id, video_id, video_title, channel_title, text_original, posted_at, last_checked_at, reply_count FROM own_comments ORDER BY posted_at DESC`

	rows, err := DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []OwnComment
	for rows.Next() {
		var c OwnComment
		var postedAtStr string
		var lastCheckedAtStr, videoTitleStr, channelTitleStr *string

		err := rows.Scan(&c.CommentID, &c.VideoID, &videoTitleStr, &channelTitleStr, &c.TextOriginal, &postedAtStr, &lastCheckedAtStr, &c.ReplyCount)
		if err != nil {
			return nil, err
		}

		if videoTitleStr != nil {
			c.VideoTitle = *videoTitleStr
		}
		if channelTitleStr != nil {
			c.ChannelTitle = *channelTitleStr
		}

		if t, err := time.Parse(time.RFC3339, postedAtStr); err == nil {
			c.PostedAt = t
		}
		if lastCheckedAtStr != nil {
			if t, err := time.Parse(time.RFC3339, *lastCheckedAtStr); err == nil {
				c.LastCheckedAt = &t
			}
		}

		comments = append(comments, c)
	}
	return comments, nil
}

func GetComment(ctx context.Context, commentID string) (*OwnComment, error) {
	query := `SELECT comment_id, video_id, video_title, channel_title, text_original, posted_at, last_checked_at, reply_count FROM own_comments WHERE comment_id = ?`

	var c OwnComment
	var postedAtStr string
	var lastCheckedAtStr, videoTitleStr, channelTitleStr *string

	err := DB.QueryRowContext(ctx, query, commentID).Scan(
		&c.CommentID, &c.VideoID, &videoTitleStr, &channelTitleStr, &c.TextOriginal, &postedAtStr, &lastCheckedAtStr, &c.ReplyCount,
	)
	if err != nil {
		return nil, err
	}

	if videoTitleStr != nil {
		c.VideoTitle = *videoTitleStr
	}
	if channelTitleStr != nil {
		c.ChannelTitle = *channelTitleStr
	}

	if t, err := time.Parse(time.RFC3339, postedAtStr); err == nil {
		c.PostedAt = t
	}
	if lastCheckedAtStr != nil {
		if t, err := time.Parse(time.RFC3339, *lastCheckedAtStr); err == nil {
			c.LastCheckedAt = &t
		}
	}

	return &c, nil
}

func DeleteComment(ctx context.Context, commentID string) error {
	_, err := DB.ExecContext(ctx, "DELETE FROM own_comments WHERE comment_id = ?", commentID)
	return err
}

func UpdateCheckedAt(ctx context.Context, commentID string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := DB.ExecContext(ctx, "UPDATE own_comments SET last_checked_at = ? WHERE comment_id = ?", now, commentID)
	return err
}
