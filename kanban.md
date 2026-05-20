# Kanban Board — YouTube Comment Dashboard

## 📋 Backlog / To Do

### Phase 4: Reply Composer (Feature 2)
- [ ] Implement `comments.insert` integration in `youtube/client.go`.
- [ ] Build HTMX reply form inline within notification cards.
- [ ] Implement error handling (403, quota exceeded, 400).
- [ ] Update DB immediately on successful reply.

### Phase 5: Full Thread Viewer (Feature 3 - Optional)
- [ ] Implement YouTube API integration for `commentThreads.list` and `channels.list`.
- [ ] Build UI to inspect the full video thread context from a reply notification.
- [ ] Implement logic to highlight the user's own comment thread in the viewer.

---

## 🏗️ In Progress

- [ ] (Empty)

---

## ✅ Done

### Phase 1: Setup & Infrastructure
- [x] Initialize Go project, install dependencies (chi, templ, air, modernc.org/sqlite).
- [x] Setup main.go with Chi router and static file server.
- [x] Create base layout.templ with Tailwind CSS, DaisyUI, HTMX, and Alpine.js.
- [x] Implement SQLite schema (own_comments, replies, oauth_tokens).
- [x] Implement YouTube OAuth 2.0 auth flow (login, callback, token storage, refresh).

### Phase 2: Core Architecture (Comment ID Registry)
- [x] Define data structs (OwnComment, Reply, NotificationItem, OAuthToken).
- [x] Implement DB CRUD: SaveComment, GetComments, GetComment, DeleteComment, UpdateCheckedAt.
- [x] Implement DB CRUD: SaveReplies (batch), MarkReplySeen, GetUnseenReplyCount.
- [x] Build YouTube API client: FetchCommentSnippet, FetchVideoSnippet, ParseCommentURL.
- [x] Build /comments page with add form and tracked comments table.
- [x] Build HTMX partials: CommentList (table + error), CommentAddForm, delete handler.
- [x] Update sidebar badge to show real comment count.

### Phase 3: Notifications Feed (Feature 1)
- [x] Integrate `FetchReplies` in `youtube/client.go` for `comments.list?parentId=` with pagination.
- [x] Implement polling logic: iterate registry, fetch replies, diff against stored, mark unseen.
- [x] Build actual notification cards in feed template (`feed.templ`).
- [x] Add HTMX polling to dashboard feed container.
- [x] Add simple in-memory quota tracking (`youtube.QuotaCounter`).
- [x] Add MarkReplySeen HTMX endpoint (`POST /partials/replies/{id}/seen`).
- [x] Wire live quota usage into navbar and dashboard stats card.
