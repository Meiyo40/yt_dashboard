package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"yt_dashboard/db"
)

const baseURL = "https://www.googleapis.com/youtube/v3"

// QuotaCounter tracks approximate API quota usage in-memory.
// Safe for concurrent access via atomic operations if needed, but
// simple increment is fine for this single-process app.
var QuotaCounter int

// CommentSnippet contains the fields we need from a comment's snippet.
type CommentSnippet struct {
	VideoID           string `json:"videoId"`
	TextOriginal      string `json:"textOriginal"`
	AuthorDisplayName string `json:"authorDisplayName"`
	PublishedAt       string `json:"publishedAt"`
}

// VideoSnippet contains the fields we need from a video's snippet.
type VideoSnippet struct {
	Title     string `json:"title"`
	ChannelID string `json:"channelId"`
}

// FetchCommentSnippet fetches a single comment by ID and returns its snippet.
func FetchCommentSnippet(ctx context.Context, accessToken, commentID string) (*CommentSnippet, error) {
	u, err := url.Parse(baseURL + "/comments")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("part", "snippet")
	q.Set("id", commentID)
	q.Set("fields", "items(snippet(videoId,textOriginal,authorDisplayName,publishedAt))")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("comments.list returned status %d", resp.StatusCode)
	}

	var result struct {
		Items []struct {
			Snippet CommentSnippet `json:"snippet"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Items) == 0 {
		return nil, fmt.Errorf("comment not found")
	}
	return &result.Items[0].Snippet, nil
}

// FetchVideoSnippet fetches video details by video ID.
func FetchVideoSnippet(ctx context.Context, accessToken, videoID string) (*VideoSnippet, error) {
	u, err := url.Parse(baseURL + "/videos")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("part", "snippet")
	q.Set("id", videoID)
	q.Set("fields", "items(snippet(title,channelId))")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("videos.list returned status %d", resp.StatusCode)
	}

	var result struct {
		Items []struct {
			Snippet VideoSnippet `json:"snippet"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Items) == 0 {
		return nil, fmt.Errorf("video not found")
	}
	return &result.Items[0].Snippet, nil
}

// FetchReplies fetches all replies for a given parent comment ID.
// It handles pagination automatically and returns a slice of Reply structs.
func FetchReplies(ctx context.Context, accessToken, parentID string) ([]db.Reply, error) {
	var allReplies []db.Reply
	pageToken := ""

	for {
		u, err := url.Parse(baseURL + "/comments")
		if err != nil {
			return nil, err
		}
		q := u.Query()
		q.Set("part", "snippet")
		q.Set("parentId", parentID)
		q.Set("maxResults", "100")
		q.Set("fields", "items(id,snippet(authorDisplayName,authorProfileImageUrl,authorChannelId,value,textDisplay,publishedAt,parentId)),nextPageToken")
		if pageToken != "" {
			q.Set("pageToken", pageToken)
		}
		u.RawQuery = q.Encode()

		req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusForbidden {
			// Comments disabled or video deleted - treat as zero replies
			QuotaCounter++
			return allReplies, nil
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("comments.list returned status %d", resp.StatusCode)
		}

		QuotaCounter++

		var result struct {
			Items []struct {
				ID      string `json:"id"`
				Snippet struct {
					AuthorDisplayName     string `json:"authorDisplayName"`
					AuthorProfileImageURL string `json:"authorProfileImageUrl"`
					AuthorChannelID       struct {
						Value string `json:"value"`
					} `json:"authorChannelId"`
					TextDisplay string `json:"textDisplay"`
					PublishedAt string `json:"publishedAt"`
					ParentID    string `json:"parentId"`
				} `json:"snippet"`
			} `json:"items"`
			NextPageToken string `json:"nextPageToken"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

		for _, item := range result.Items {
			publishedAt, _ := time.Parse(time.RFC3339, item.Snippet.PublishedAt)
			reply := db.Reply{
				ID:                    item.ID,
				ParentID:              item.Snippet.ParentID,
				AuthorDisplayName:     item.Snippet.AuthorDisplayName,
				AuthorProfileImageURL: item.Snippet.AuthorProfileImageURL,
				AuthorChannelID:       item.Snippet.AuthorChannelID.Value,
				TextDisplay:           item.Snippet.TextDisplay,
				PublishedAt:           publishedAt,
				Seen:                  false,
			}
			allReplies = append(allReplies, reply)
		}

		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}

	return allReplies, nil
}

// ParseCommentURL extracts videoId and commentId from a YouTube comment URL.
// Supports formats:
//
//	https://www.youtube.com/watch?v=VIDEO_ID&lc=COMMENT_ID
//	https://youtu.be/VIDEO_ID?lc=COMMENT_ID
func ParseCommentURL(rawURL string) (videoID, commentID string, ok bool) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", "", false
	}

	// youtube.com/watch?v=...&lc=...
	if strings.Contains(u.Host, "youtube.com") || strings.Contains(u.Host, "youtu.be") {
		// Extract video ID
		if strings.Contains(u.Host, "youtu.be") {
			videoID = strings.TrimPrefix(u.Path, "/")
		} else {
			videoID = u.Query().Get("v")
		}

		// Extract comment ID from lc parameter
		commentID = u.Query().Get("lc")

		if videoID != "" && commentID != "" {
			return videoID, commentID, true
		}
	}

	return "", "", false
}
