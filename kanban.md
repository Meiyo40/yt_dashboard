# Kanban Board — YouTube Comment Dashboard

## 📋 Backlog / To Do

### Phase 1: Setup & Infrastructure
- [ ] Initialize Go project (`go mod init`) and install dependencies (`chi`, `templ`, `air`, `modernc.org/sqlite`).
- [ ] Setup `main.go` with Chi router and serve static files.
- [ ] Create base `layout.templ` with Tailwind CSS, DaisyUI, HTMX, and Alpine.js.
- [ ] Implement SQLite database initialization and schema creation (Comment Registry, Replies, OAuth Tokens).
- [ ] Implement YouTube OAuth 2.0 authentication flow (login, callback, token storage, refresh token mechanism).

### Phase 2: Core Architecture (Comment ID Registry)
- [ ] Define `OwnComment` struct and implement DB CRUD operations.
- [ ] Define `Reply` struct and implement DB CRUD operations.
- [ ] Build UI to manually add a tracked comment to the registry (e.g., paste a YouTube comment URL).

### Phase 3: Notifications Feed (Feature 1)
- [ ] Implement YouTube API integration for `comments.list` to fetch replies for a given `parentId`.
- [ ] Implement background polling strategy (iterate registry, fetch new replies, update `lastCheckedAt`).
- [ ] Implement diffing logic to mark new replies as "unseen" and update `replyCount`.
- [ ] Build Dashboard UI (HTMX feed) to display `NotificationItem` (reply + original comment context).

### Phase 4: Reply Composer (Feature 2)
- [ ] Implement YouTube API integration for `comments.insert` to post a reply.
- [ ] Build HTMX form inline within the Notification feed to compose a reply.
- [ ] Implement error handling for reply submission (403 forbidden, quota exceeded, 400 bad request).
- [ ] Update local database immediately upon successful reply insertion.

### Phase 5: Full Thread Viewer (Feature 3 - Optional)
- [ ] Implement YouTube API integration for `commentThreads.list` and `channels.list` (to get own channel ID).
- [ ] Build UI to inspect the full video thread context from a reply notification.
- [ ] Implement logic to highlight the user's own comment thread in the viewer.

---

## 🏗️ In Progress

- [ ] (Empty)

---

## ✅ Done

- [x] Project specifications and architecture defined.
