package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const baseURL = "https://www.googleapis.com/youtube/v3"

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
