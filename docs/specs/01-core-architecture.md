# Core Architecture & Foundation

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
