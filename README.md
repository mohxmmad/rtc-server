# RTC Server — Live Collaboration Server

<p align="center">
  <video src="./RTC demo.mp4" width="100%" controls autoplay muted loop></video>
</p>

<p align="center">
  <b>🎬 Watch the live demo above — two editors, one room, instant sync</b><br/>
  <sub>GRTC (Godot) + Unity • Session-authenticated WebSocket rooms • File & live transform sync</sub>
</p>

> **GRTC is live.** This is no longer just an OAuth helper — it's a real-time collaboration backend. Two users clone the same repo, log in with GitHub, join the same `project:<hash>` room (derived from the normalized git remote URL), and see each other's edits instantly: file updates and live node transforms (`Cube` moving in `GRTC.tscn`) without pressing Save.

## Demo

**`RTC demo.mp4`** is the main attraction. The video at the top shows two Godot instances editing the same project live. Play it directly on GitHub — the file is tracked at `./RTC demo.mp4` (3.5 MB).

If the video doesn't autoplay, click it or **[Download / Open RTC demo.mp4](./RTC%20demo.mp4)**.

## Features

*   **GitHub OAuth (session-based)** + session restore via `X-Session-ID` and token lookup
*   **Live WebSocket rooms** — `ws://localhost:8000/ws?session_id=...&room=project:<hash>&client=godot` — auto-joins, presence via `/ws/online-users`
*   **File sync** — `publish` `file_created / file_updated / file_deleted` with base64 content, ordered unified log on client
*   **Live scene sync** — `live_node` `T:x,y,z|R:x,y,z|S:x,y,z` applied directly to the edited scene in memory (no save needed for transforms)
*   **User / Project CRUD** + **PostgreSQL** (users, projects, github_data, collaborators, activities)
*   **Middleware** — logging, CORS, security headers, rate limit 60/min burst 10

## Quick Start

```bash
# env: DATABASE_URL, PORT=8000, SECRET_KEY, GITHUB_CLIENT_ID/SECRET, GITHUB_CALLBACK_URL
go run .
# health
curl http://localhost:8000/health # -> OK
# ws
ws://localhost:8000/ws?session_id=<session_id>&room=project:<hash>&client=godot
```

## Tech Stack

**Language**: Go • **Auth**: GitHub OAuth • **DB**: PostgreSQL • **HTTP**: net/http + gorilla/mux/handlers • **Realtime**: gorilla/websocket

## Project Structure

<img src="screenshots/structure.png" alt="Structure" width="600">

## WebSocket Protocol (Unity + Godot shared)

Connect:
```text
ws://localhost:8000/ws?session_id=<session_id>&project_id=<project_uuid>&client=unity
ws://localhost:8000/ws?session_id=<session_id>&room=project:<hash>&client=godot
```

Client → Server:
```json
{ "type": "auth", "session_id": "...", "client": "godot" }
{ "type": "join", "room": "project:abc" }
{ "type": "publish", "room": "project:abc", "event": "file_updated", "data": { "path": "GRTC.tscn", "file_content": "base64..." } }
{ "type": "publish", "room": "project:abc", "event": "live_node", "data": { "path": "GRTC.tscn", "node_path": "Cube", "state": "T:0,1,0|R:0,0,0|S:1,1,1" } }
{ "type": "sync_request" }
{ "type": "ping" }
```

Server → Client: `welcome`, `auth_ok`, `room_joined`, `event`, `sync_state`, `pong`, `error`. Room is the normalized git remote (`https://github.com/...` without `.git`, lowercased) → same clone = same room across accounts.

Presence:
```
GET /ws/online-users
GET /ws/user-status?user_id=<uuid>
GET /ws?session_id=...&room=...
```

## Screenshots

<img src="screenshots/oauth.png" alt="oauth" width="600">
<img src="screenshots/new_project.png" alt="new_project" width="600">
<img src="screenshots/logs.png" alt="logs" width="600">
<img src="screenshots/collaboration.png" alt="collaboration" width="600">

---

**Licensed under MIT — github.com/Mohammad-416 • 2025-2026**
