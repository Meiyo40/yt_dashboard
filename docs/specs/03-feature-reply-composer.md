# Feature 2: Reply Composer

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
