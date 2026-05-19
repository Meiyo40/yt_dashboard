package db

import (
	"time"
)

// OwnComment represents a comment posted by the user, tracked in our registry.
type OwnComment struct {
	CommentID     string     `json:"commentId"`
	VideoID       string     `json:"videoId"`
	VideoTitle    string     `json:"videoTitle,omitempty"`
	ChannelTitle  string     `json:"channelTitle,omitempty"`
	TextOriginal  string     `json:"textOriginal"`
	PostedAt      time.Time  `json:"postedAt"`
	LastCheckedAt *time.Time `json:"lastCheckedAt,omitempty"`
	ReplyCount    int        `json:"replyCount"`
}

// Reply represents a reply on our tracked comment.
type Reply struct {
	ID                    string    `json:"id"`
	ParentID              string    `json:"parentId"` // References OwnComment.CommentID
	AuthorDisplayName     string    `json:"authorDisplayName"`
	AuthorProfileImageURL string    `json:"authorProfileImageUrl"`
	AuthorChannelID       string    `json:"authorChannelId"`
	TextDisplay           string    `json:"textDisplay"`
	PublishedAt           time.Time `json:"publishedAt"`
	Seen                  bool      `json:"seen"`
}

// NotificationItem pairs a reply with its parent comment context.
type NotificationItem struct {
	Reply      Reply      `json:"reply"`
	OwnComment OwnComment `json:"ownComment"`
	Seen       bool       `json:"seen"`
}

// OAuthToken represents the OAuth2 token structure we store.
type OAuthToken struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	TokenType    string    `json:"tokenType"`
	Expiry       time.Time `json:"expiry"`
}
