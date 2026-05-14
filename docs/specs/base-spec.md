# YouTube Comment Dashboard — API Specification (v2)

A technical specification for tracking replies to your own comments posted on other channels,
and responding to those replies from a custom dashboard, using the YouTube Data API v3.

---

## Overview & Core Problem

The goal is to:

1. **See replies** to comments you posted on other people's videos
2. **Reply back** to those replies from the dashboard
3. **Optionally** open the full comment thread of the video when inspecting a reply

### Critical API Limitation

The YouTube Data API v3 has **no endpoint to list all comments posted by a given user**,
even with the user's own OAuth token. There is no `commentThreads.list?authorId=me` filter.

This means there is no direct "fetch my comments on other channels" query. The only viable
approaches are:

- **Approach A (recommended):** Persist your own comment IDs client-side, then poll for
  replies using `comments.list?parentId={your_comment_id}`.
- **Approach B (bootstrap):** Use `commentThreads.list?allThreadsRelatedToChannelId=` on
  each channel you know you've commented on — expensive and impractical at scale.
- **Approach C (unofficial, fragile):** Scrape `myactivity.google.com/page?page=youtube_comments`
  — not part of any official API, may break without notice. Not recommended for production.

This spec uses **Approach A** as the primary strategy.

---

## Authentication

### OAuth 2.0 Scopes

| Scope | Used for |
|-------|----------|
| `https://www.googleapis.com/auth/youtube.readonly` | Reading comment threads and replies |
| `https://www.googleapis.com/auth/youtube.force-ssl` | Posting replies (`comments.insert`) |

**Flow type:** Authorization Code + PKCE (for SPAs) or Authorization Code (for server-side apps).

**Token handling:**

- Access tokens expire after 1 hour — store and use the `refresh_token` for silent renewal.
- Never store tokens in `localStorage` (sandboxed iframes block it). Use secure cookies
  (server-side) or in-memory variables (client-side SPA).

**Google Cloud Console setup:**

1. Create a project and enable the **YouTube Data API v3**
2. Create OAuth 2.0 credentials (type: Web Application or Desktop App)
3. Add your redirect URI to the authorized redirect list

---

## Quota Budget

Default daily allowance: **10,000 units** per Google Cloud project.

| Operation | Method | Quota cost |
|-----------|--------|-----------|
| List replies for a comment | `comments.list` | 1 unit/request |
| Post a reply | `comments.insert` | 50 units |
| List threads for a video | `commentThreads.list` | 1 unit/request |

**Practical budget at 10,000 units/day:**

- ~10,000 reply-listing requests (very cheap)
- ~200 replies posted before hitting the cap

---

## Core Architecture: Comment ID Registry

Since the API cannot enumerate your comments, the dashboard must maintain a local
**Comment ID Registry** — a persisted list of comment IDs for comments you have posted.

```ts
type OwnComment = {
  commentId: string;       // The comment resource ID returned by comments.insert
  videoId: string;         // The video the comment was posted on
  videoTitle?: string;     // Optional: store for display purposes
  channelTitle?: string;   // Optional: store for display purposes
  textOriginal: string;    // Your comment text
  postedAt: string;        // ISO 8601 timestamp
  lastCheckedAt?: string;  // When replies were last fetched
  replyCount: number;      // Last known reply count
};
```

**Populating the registry:**

- When you post a new comment from the dashboard, store the returned `comment.id` immediately.
- Optionally allow manual import: paste a YouTube comment URL and extract the comment ID
  from the `lc=` query parameter (e.g. `?lc=UgxABC123`).

---

## Feature 1: Notifications Feed — Replies to Your Comments

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

---

## Feature 2: Reply Composer

Post a reply from the dashboard directly to a reply in your thread.

### Important: YouTube reply threading is flat

YouTube does not support nested replies. When someone replies to your comment,
their reply's `parentId` is your original comment's `id` — not the reply's own `id`.
When you reply back, you always reply to the **original top-level comment thread**.

### 2.1 Post a Reply

**Endpoint:** `POST https://www.googleapis.com/youtube/v3/comments`

**Query parameter:**

| Parameter | Value |
|-----------|-------|
| `part` | `snippet` |

**Authorization:** `youtube.force-ssl` scope required.

**Request body:**

```json
{
  "snippet": {
    "parentId": "{your_original_comment_id}",
    "textOriginal": "Thanks for your reply! ..."
  }
}
```

> **Note:** `parentId` is always the **top-level comment ID** (your original comment),
> never the ID of the individual reply you're responding to. To address someone
> specifically, mention their name in the text (e.g. `@Username`).

**Response:** Returns the created `comment` resource with its assigned `id` and full `snippet`.

**After success:**

- Append the returned comment to the local replies list immediately (no re-fetch needed).
- Update `OwnComment.replyCount` in the registry.
- Quota: deduct 50 units from daily counter.

**Error cases to handle:**

| HTTP status | Error reason | Handling |
|------------|-------------|---------|
| `403 forbidden` | `commentsDisabled` | Show warning: "Replies are disabled on this video" |
| `403 forbidden` | `videoNotFound` or deleted | Show warning: "The video no longer exists" |
| `400 badRequest` | Empty or too-long text | Validate before sending (max ~10,000 chars) |
| `403 quotaExceeded` | Daily quota hit | Block writes; show quota warning |
| `401 unauthorized` | Token expired | Trigger silent token refresh, then retry |

---

## Feature 3: Full Video Thread Viewer (Optional)

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

---

## Data Model (Client-Side)

### `OwnComment` (Comment ID Registry entry)

```ts
type OwnComment = {
  commentId: string;
  videoId: string;
  videoTitle?: string;
  channelTitle?: string;
  textOriginal: string;
  postedAt: string;             // ISO 8601
  lastCheckedAt?: string;       // ISO 8601
  replyCount: number;
};
```

### `Reply`

```ts
type Reply = {
  id: string;
  parentId: string;             // Always your OwnComment.commentId
  authorDisplayName: string;
  authorProfileImageUrl: string;
  authorChannelId: string;
  textDisplay: string;
  publishedAt: string;          // ISO 8601
  seen: boolean;                // Local flag — not from API
};
```

### `NotificationItem`

```ts
type NotificationItem = {
  reply: Reply;
  ownComment: OwnComment;       // The comment being replied to
  seen: boolean;
};
```

---

## API Endpoint Reference Summary

| Feature | Method | Endpoint | Auth scope | Quota cost |
|---------|--------|----------|-----------|-----------|
| Fetch replies to your comment | `comments.list` | `/youtube/v3/comments?parentId=` | readonly | 1 |
| Post a reply | `comments.insert` | `POST /youtube/v3/comments` | force-ssl | 50 |
| Get your channel ID | `channels.list` | `/youtube/v3/channels?mine=true` | readonly | 1 |
| List all threads on a video | `commentThreads.list` | `/youtube/v3/commentThreads?videoId=` | readonly | 1 |

---

## Edge Cases & Constraints

- **No API to list your own comments.** The registry is mandatory. Without it, you cannot
  know which comment IDs to poll. There is no `commentThreads.list?author=me` filter.
- **Flat reply threading.** All replies in a thread share the same `parentId` (the top-level
  comment). There is no concept of "reply to a reply" — `parentId` must always point to
  the top-level comment, never to another reply.
- **Deleted or private videos.** A `comments.list` call on a `parentId` whose video was
  deleted will return `403` or an empty response. Remove that entry from the registry or
  mark it as `archived`.
- **Comment deletion by video owner.** A third party can delete your comment from their video.
  A `comments.list?parentId={deleted_id}` will return `404`. Handle gracefully and mark as
  deleted in registry.
- **Pagination is required.** All `list` responses may include a `nextPageToken`. Always
  iterate until exhausted if you want all replies.
- **Rate limiting.** Beyond daily quota, YouTube may return `429` or `503` on burst requests.
  Implement exponential backoff with jitter.
- **`replies` in `commentThreads.list` is partial.** Max 5 replies inline. Use
  `comments.list?parentId=` for the full chain.

---

## Relevant Documentation

- [YouTube Data API v3 Overview](https://developers.google.com/youtube/v3/getting-started)
- [comments.list](https://developers.google.com/youtube/v3/docs/comments/list)
- [comments.insert](https://developers.google.com/youtube/v3/docs/comments/insert)
- [commentThreads.list](https://developers.google.com/youtube/v3/docs/commentThreads/list)
- [channels.list](https://developers.google.com/youtube/v3/docs/channels/list)
- [Implementation: Comments](https://developers.google.com/youtube/v3/guides/implementation/comments)
- [Quota Calculator](https://developers.google.com/youtube/v3/determine_quota_cost)
- [OAuth 2.0 PKCE Flow](https://developers.google.com/identity/protocols/oauth2/native-app)
