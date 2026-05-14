# YouTube Comment Dashboard — API Specification

A technical specification for building a custom YouTube comment management dashboard using the YouTube Data API v3. Stack-agnostic: covers only API endpoints, OAuth flow, data models, and behavioral contracts.

---

## Overview

This dashboard provides a single interface to:

1. **Notifications feed** — see all latest comments and replies received across all your videos
2. **Reply composer** — respond to any comment directly from the dashboard
3. **Video thread viewer** — (optional) open the full comment thread of a given video

All API calls are authenticated with OAuth 2.0 and require the channel owner's credentials.

---

## Authentication

### OAuth 2.0 Flow

The dashboard requires OAuth 2.0 authorization. Read-only operations (listing comments) use the read scope; write operations (posting replies) require the broader scope.

| Scope                                               | Used for                            |
| --------------------------------------------------- | ----------------------------------- |
| `https://www.googleapis.com/auth/youtube.readonly`  | Reading comments and channel data   |
| `https://www.googleapis.com/auth/youtube.force-ssl` | Posting replies (`comments.insert`) |

**Flow type:** Authorization Code (for server-side apps) or PKCE Authorization Code (for SPAs / native apps).

**Token storage:** Access tokens expire after 1 hour. Store and refresh using the `refresh_token` returned during initial authorization. Never store tokens in `localStorage` on the web — use secure cookies (server-side) or in-memory (client-side SPA).

**Required Google Cloud setup:**

- Create a project in [Google Cloud Console](https://console.cloud.google.com)
- Enable the **YouTube Data API v3**
- Create OAuth 2.0 credentials (type: Web Application or Desktop App)
- Add your redirect URI to the allowed list

---

## Quota Budget

Every API call consumes quota units from a default daily allowance of **10,000 units** per project.

| Operation                    | Method                  | Quota cost     |
| ---------------------------- | ----------------------- | -------------- |
| List comment threads         | `commentThreads.list`   | 1 unit/request |
| List replies                 | `comments.list`         | 1 unit/request |
| Post a reply                 | `comments.insert`       | 50 units       |
| Post a new top-level comment | `commentThreads.insert` | 50 units       |
| Update a comment             | `comments.update`       | 50 units       |
| Delete a comment             | `comments.delete`       | 50 units       |

**Practical limits at 10,000 units/day:**

- ~10,000 listing requests (read-heavy usage is essentially free)
- ~200 replies posted per day before hitting the cap

To increase the daily quota, submit a [quota extension request](https://support.google.com/youtube/contact/yt_api_form) to Google. Approval requires a compliance audit.

---

## Feature 1: Notifications Feed

Fetches all recent comments and replies received on the authenticated channel.

### 1.1 Fetch All Comment Threads for the Channel

Retrieve every comment thread associated with the channel (both channel-level comments and all video comments).

**Endpoint:** `GET https://www.googleapis.com/youtube/v3/commentThreads`

**Parameters:**

| Parameter                      | Value               | Notes                                              |
| ------------------------------ | ------------------- | -------------------------------------------------- |
| `part`                         | `snippet,replies`   | Returns top-level comment + up to 5 inline replies |
| `allThreadsRelatedToChannelId` | `{your_channel_id}` | Fetches threads across all videos of the channel   |
| `order`                        | `time`              | Sort by newest first (`time` or `relevance`)       |
| `maxResults`                   | `20` to `100`       | Items per page (max 100)                           |
| `pageToken`                    | `{nextPageToken}`   | Pagination cursor from previous response           |
| `moderationStatus`             | `published`         | Only show published comments (omit for all)        |

**Authorization:** Required (read scope sufficient).

**Sample request:**

```
GET https://www.googleapis.com/youtube/v3/commentThreads
  ?part=snippet,replies
  &allThreadsRelatedToChannelId=UC_XXXXXXXXXXXXXXXXXXXX
  &order=time
  &maxResults=50
Authorization: Bearer {access_token}
```

**Response object — `commentThread` resource:**

```json
{
  "kind": "youtube#commentThread",
  "id": "string",
  "snippet": {
    "channelId": "string",
    "videoId": "string",
    "topLevelComment": { "...comment resource..." },
    "canReply": true,
    "totalReplyCount": 3,
    "isPublic": true
  },
  "replies": {
    "comments": [ "...up to 5 comment resources..." ]
  }
}
```

**Key fields to extract for the notification card:**

| Field path                                          | Usage                                        |
| --------------------------------------------------- | -------------------------------------------- |
| `id`                                                | Thread ID (used as `parentId` when replying) |
| `snippet.videoId`                                   | Link back to the video                       |
| `snippet.topLevelComment.snippet.authorDisplayName` | Comment author                               |
| `snippet.topLevelComment.snippet.textDisplay`       | Comment text                                 |
| `snippet.topLevelComment.snippet.publishedAt`       | Publication timestamp (ISO 8601)             |
| `snippet.topLevelComment.snippet.likeCount`         | Like count                                   |
| `snippet.totalReplyCount`                           | Number of replies                            |
| `replies.comments[]`                                | Inline replies (partial — up to 5)           |

### 1.2 Polling for New Comments

The YouTube Data API v3 does **not** provide webhooks or push notifications for new comments. You must poll at regular intervals.

**Recommended polling strategy:**

- On first load: fetch the last N threads sorted by `time`.
- On subsequent polls: store the `publishedAt` timestamp of the most recent comment seen. After each poll, only display threads newer than the stored timestamp as "new."
- **Polling interval:** Minimum 60 seconds recommended. Do not hammer the API to avoid quota exhaustion.
- Alternatively, use `publishedBefore` / `publishedAfter` parameters (not natively supported on `commentThreads.list`) — instead, filter client-side by `snippet.topLevelComment.snippet.publishedAt`.

---

## Feature 2: Reply Composer

Allows the channel owner to post a reply to any top-level comment thread.

### 2.1 Post a Reply

**Endpoint:** `POST https://www.googleapis.com/youtube/v3/comments`

**Query parameter:**

| Parameter | Value     |
| --------- | --------- |
| `part`    | `snippet` |

**Authorization:** Required — must use `youtube.force-ssl` scope.

**Request body:**

```json
{
  "snippet": {
    "parentId": "{commentThread.id}",
    "textOriginal": "Your reply text here."
  }
}
```

**Key fields:**

| Field                  | Description                                                 |
| ---------------------- | ----------------------------------------------------------- |
| `snippet.parentId`     | The `id` of the `commentThread` resource being replied to   |
| `snippet.textOriginal` | Plain-text content of the reply (supports some HTML subset) |

**Response:** Returns the created `comment` resource with its assigned `id` and `snippet`.

**Error cases to handle:**

| HTTP status               | Error reason                | Handling                                             |
| ------------------------- | --------------------------- | ---------------------------------------------------- |
| `403 forbidden`           | `canReply: false` on thread | Disable reply button if `snippet.canReply === false` |
| `403 forbidden`           | `commentsDisabled`          | Comments turned off for the video                    |
| `400 badRequest`          | Empty or invalid text       | Validate non-empty input before sending              |
| `429 / 403 quotaExceeded` | Daily quota hit             | Show quota warning; block further writes             |

### 2.2 UI Contract for the Reply Composer

- Check `snippet.canReply` from the `commentThread` resource before rendering the reply button.
- Show a character count (YouTube allows up to ~10,000 characters per comment).
- After a successful `comments.insert`, append the returned `comment` resource to the local replies list without re-fetching.
- Quota: each reply costs **50 units** — display a running counter if quota management is desired.

---

## Feature 3: Full Video Thread Viewer (Optional)

Opens the complete list of comment threads for a specific video, with full reply chains.

### 3.1 Fetch Comment Threads for a Video

**Endpoint:** `GET https://www.googleapis.com/youtube/v3/commentThreads`

**Parameters:**

| Parameter    | Value                 | Notes                             |
| ------------ | --------------------- | --------------------------------- |
| `part`       | `snippet,replies`     | Include inline replies            |
| `videoId`    | `{video_id}`          | The video whose comments to fetch |
| `order`      | `time` or `relevance` | Sort order                        |
| `maxResults` | Up to `100`           | Items per page                    |
| `pageToken`  | `{nextPageToken}`     | Cursor for pagination             |

**Sample request:**

```
GET https://www.googleapis.com/youtube/v3/commentThreads
  ?part=snippet,replies
  &videoId=dQw4w9WgXcQ
  &order=time
  &maxResults=100
Authorization: Bearer {access_token}
```

### 3.2 Fetch All Replies for a Thread

When a thread has more replies than the inline `replies.comments[]` array (i.e., `totalReplyCount > replies.comments.length`), fetch the full reply chain separately.

**Endpoint:** `GET https://www.googleapis.com/youtube/v3/comments`

**Parameters:**

| Parameter    | Value                | Notes                             |
| ------------ | -------------------- | --------------------------------- |
| `part`       | `snippet`            |                                   |
| `parentId`   | `{commentThread.id}` | The thread whose replies to fetch |
| `maxResults` | Up to `100`          |                                   |
| `pageToken`  | `{nextPageToken}`    | For pagination                    |

**Sample request:**

```
GET https://www.googleapis.com/youtube/v3/comments
  ?part=snippet
  &parentId=THREAD_ID
  &maxResults=100
Authorization: Bearer {access_token}
```

---

## Data Model (Client-Side)

Suggested normalized client-side model to power the dashboard UI.

### `CommentThread`

```ts
type CommentThread = {
  id: string; // YouTube thread ID (used as parentId for replies)
  videoId: string;
  topLevelComment: Comment;
  totalReplyCount: number;
  canReply: boolean;
  isPublic: boolean;
  replies: Comment[]; // Partial (up to 5) from commentThreads.list
  repliesFullyLoaded: boolean; // True once comments.list has been called
};
```

### `Comment`

```ts
type Comment = {
  id: string;
  parentId?: string; // Only set for replies
  authorDisplayName: string;
  authorProfileImageUrl: string;
  authorChannelId: string;
  textDisplay: string; // HTML-formatted text
  textOriginal: string; // Plain text (for editing)
  likeCount: number;
  publishedAt: string; // ISO 8601
  updatedAt: string; // ISO 8601
};
```

---

## API Endpoint Reference Summary

| Feature                   | Method                  | Endpoint                                                   | Auth scope | Quota cost |
| ------------------------- | ----------------------- | ---------------------------------------------------------- | ---------- | ---------- |
| List all channel threads  | `commentThreads.list`   | `/youtube/v3/commentThreads?allThreadsRelatedToChannelId=` | readonly   | 1          |
| List video threads        | `commentThreads.list`   | `/youtube/v3/commentThreads?videoId=`                      | readonly   | 1          |
| List replies for a thread | `comments.list`         | `/youtube/v3/comments?parentId=`                           | readonly   | 1          |
| Post a reply              | `comments.insert`       | `POST /youtube/v3/comments`                                | force-ssl  | 50         |
| Post a top-level comment  | `commentThreads.insert` | `POST /youtube/v3/commentThreads`                          | force-ssl  | 50         |
| Update a comment          | `comments.update`       | `PUT /youtube/v3/comments`                                 | force-ssl  | 50         |
| Delete a comment          | `comments.delete`       | `DELETE /youtube/v3/comments`                              | force-ssl  | 50         |

---

## Edge Cases & Constraints

- **`replies` in `commentThreads.list` is partial.** It contains at most 5 replies. Always check `totalReplyCount > replies.comments.length` and lazy-load the full chain via `comments.list` when the user expands a thread.
- **`allThreadsRelatedToChannelId` vs `channelId`.** Use `allThreadsRelatedToChannelId` for the notifications feed — it returns threads from all videos. `channelId` returns only channel-level comments (not video comments).
- **Pagination is mandatory.** All `list` methods are paginated. Iterate `nextPageToken` to retrieve all results. There is no `offset` — always use the cursor.
- **`order=time` is not guaranteed to be strict.** YouTube may return some out-of-order results near the page boundary. Sort client-side by `publishedAt` as a secondary safety measure.
- **Comments from private or deleted videos** may return `403` or be absent from `allThreadsRelatedToChannelId` results. Handle gracefully.
- **Rate limit vs quota limit.** Beyond the daily 10,000-unit quota, YouTube may also rate-limit burst requests. Implement exponential backoff on `429` and `503` responses.
- **No server-push / webhooks.** Polling is the only option. For near-real-time notifications, consider a background job that polls every 60 seconds and marks delta comments as "unseen."

---

## Relevant Documentation Links

- [YouTube Data API v3 Overview](https://developers.google.com/youtube/v3/getting-started)
- [commentThreads Resource](https://developers.google.com/youtube/v3/docs/commentThreads)
- [commentThreads.list](https://developers.google.com/youtube/v3/docs/commentThreads/list)
- [comments Resource](https://developers.google.com/youtube/v3/docs/comments)
- [comments.list](https://developers.google.com/youtube/v3/docs/comments/list)
- [comments.insert](https://developers.google.com/youtube/v3/docs/comments/insert)
- [Implementation: Comments](https://developers.google.com/youtube/v3/guides/implementation/comments)
- [Quota Calculator](https://developers.google.com/youtube/v3/determine_quota_cost)
- [OAuth 2.0 for Web Server Applications](https://developers.google.com/identity/protocols/oauth2/web-server)
