# Feature 3: Full Video Thread Viewer (Optional)

When inspecting a reply notification, optionally display the full thread context
of the video — all top-level comments on that video.

### 3.1 Fetch Comment Threads for a Video

**Endpoint:** `GET https://www.googleapis.com/youtube/v3/commentThreads`

**Parameters:**

| Parameter | Value | Notes |
|-----------|-------|-------|
| `part` | `snippet,replies` | Top-level comment + up to 5 inline replies |
| `videoId` | `{video_id}` | From your `OwnComment.videoId` |
| `order` | `time` or `relevance` | |
| `maxResults` | Up to `100` | Items per page |
| `pageToken` | `{nextPageToken}` | Pagination cursor |

**Note:** `snippet.replies` returns at most 5 replies. To show the full reply chain
for your own comment specifically, call `comments.list?parentId={your_comment_id}` as
described in Feature 1 — this is more efficient than loading all threads.

### 3.2 Highlight Your Own Comment

When rendering video threads, highlight the `commentThread` whose
`snippet.topLevelComment.snippet.authorChannelId.value` matches your channel ID.
Your channel ID is obtainable via:

```
GET https://www.googleapis.com/youtube/v3/channels?part=snippet&mine=true
Authorization: Bearer {access_token}
```

Response field: `items[0].id` → your channel ID.
