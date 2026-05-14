# Feature 1: Notifications Feed — Replies to Your Comments

Polls for new replies to each of your tracked comments.

### 1.1 Fetch Replies for a Specific Comment

**Endpoint:** `GET https://www.googleapis.com/youtube/v3/comments`

**Parameters:**

| Parameter | Value | Notes |
|-----------|-------|-------|
| `part` | `snippet` | Returns author, text, timestamps |
| `parentId` | `{your_comment_id}` | The ID of *your* top-level comment |
| `maxResults` | Up to `100` | Items per page |
| `pageToken` | `{nextPageToken}` | Pagination cursor |
| `textFormat` | `plainText` or `html` | Format of `textDisplay` field |

**Authorization:** Read scope sufficient.

**Sample request:**

```
GET https://www.googleapis.com/youtube/v3/comments
  ?part=snippet
  &parentId=UgxABC123XYZ
  &maxResults=100
  &textFormat=plainText
Authorization: Bearer {access_token}
```

**Response — `comment` resource:**

```json
{
  "kind": "youtube#comment",
  "id": "string",
  "snippet": {
    "parentId": "string",
    "textDisplay": "string",
    "textOriginal": "string",
    "authorDisplayName": "string",
    "authorProfileImageUrl": "string",
    "authorChannelId": { "value": "string" },
    "likeCount": 0,
    "publishedAt": "2026-05-14T19:00:00.000Z",
    "updatedAt": "2026-05-14T19:00:00.000Z"
  }
}
```

**Key fields for the notification card:**

| Field path | Usage |
|-----------|-------|
| `id` | Reply ID (used as `parentId` if you reply to a reply — note: YouTube flattens replies, all replies go to the same thread) |
| `snippet.parentId` | ID of your original comment |
| `snippet.authorDisplayName` | Who replied |
| `snippet.authorProfileImageUrl` | Their avatar |
| `snippet.textDisplay` | Reply text |
| `snippet.publishedAt` | When it was posted |

### 1.2 Polling Strategy

Since the API provides no push notifications, implement client-side polling.

**Algorithm:**

```
On startup:
  For each OwnComment in registry:
    Fetch comments.list?parentId={commentId}
    Store replies locally indexed by reply.id
    Mark replies newer than lastCheckedAt as "unseen"
    Update OwnComment.lastCheckedAt = now()
    Update OwnComment.replyCount = total replies received

On interval (every 60–120 seconds):
  For each OwnComment in registry (prioritize recently active ones):
    Fetch new replies page
    Diff against stored replies (compare by id)
    New entries → push to notification feed as unseen
    Update registry
```

**Optimization:** To avoid polling all tracked comments equally, sort the registry by
`lastSeenReplyAt` descending — comments with recent activity are more likely to get
new replies soon. Only poll the top N (e.g. top 20) on each tick.
