<div align="center">

# RTC Server

### Live Collaboration Backend for Godot & Unity

[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go)](https://go.dev)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat&logo=postgresql)](https://www.postgresql.org)
[![WebSocket](https://img.shields.io/badge/Realtime-WebSocket-010101?style=flat&logo=socketdotio)](https://github.com/gorilla/websocket)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](./LICENSE)

**Clone the same repo. Login with GitHub. Edit together — live.**

*Session-authenticated rooms • File sync • Live scene transforms • No save needed*

</div>

---

### 🎬 Live Demo — Two Editors, One Project

<p align="center">
  <video src="https://github.com/Mohammad-416/rtc-server/raw/refs/heads/main/RTC%20demo.mp4"
         width="100%"
         controls>
  </video>
</p>

<p align="center">
  <b>▶️ Live collaboration demo</b> · 3.5 MB
</p>


> The video shows two Godot instances in the same `project:<hash>` room moving the **Cube** in `GRTC.tscn`. The change streams over WebSocket and applies instantly on the other editor.

---

## What it is

**RTC Server** is a Go backend that turns any Git repo into a live collaboration room.

*   **GitHub OAuth** — login, get a session (`X-Session-ID`), restore it later
*   **Rooms by repo** — the room ID is `project:<hash>` of the normalized git remote URL (`https://github.com/user/repo` lowercased, without `.git`). Every clone of the same repo joins the same room automatically, even with different GitHub accounts
*   **Live sync** — file updates (`file_created` / `file_updated` / `file_deleted` as base64) and live node transforms (`live_node` `T:x,y,z|R:x,y,z|S:x,y,z`) are broadcast ordered to the room
*   **Presence** — who is online in the room
*   **Persisted** — users, projects, GitHub tokens, collaborators and activity history in PostgreSQL

Works with the **GRTC Godot addon** (`~/grtc`) and a Unity client using the same JSON protocol.

## Features

| Area | Details |
|------|---------|
| **Auth** | GitHub OAuth, session manager (24h), `X-Session-ID` / `Authorization: Bearer` |
| **Realtime** | `ws://host:8000/ws?session_id=...&room=project:<hash>&client=godot` — `auth`, `join`, `publish`, `sync_request`, `ping` ↔ `welcome`, `auth_ok`, `room_joined`, `event`, `sync_state`, `pong` |
| **Live** | `live_node` streams Spatial transforms directly to the edited scene in memory; file changes stream as base64 and auto-apply with ordered unified log |
| **REST** | `GET /health`, `GET /db/users*`, `GET /api/activity/*`, `GET /ws/online-users`, `GET /ws/user-status` — see [`api.md`](./api.md) |
| **Safety** | Request logger, CORS, security headers, rate limit 60/min burst 10, `http.Hijacker` support for WebSocket upgrade |

## Quick Start

```bash
# .env
DATABASE_URL=postgres://user:pass@host:5432/db?sslmode=require
PORT=8000
SECRET_KEY=...
GITHUB_CLIENT_ID=...
GITHUB_CLIENT_SECRET=...
GITHUB_CALLBACK_URL=http://localhost:8000/github/callback

go run .
curl http://localhost:8000/health # OK
```

**Connect:**
```text
ws://localhost:8000/ws?session_id=<session_id>&room=project:<hash>&client=godot
ws://localhost:8000/ws?session_id=<session_id>&project_id=<uuid>&client=unity
```

## WebSocket Protocol

**Client → Server**
```json
{ "type": "auth", "session_id": "...", "client": "godot" }
{ "type": "join", "room": "project:abc" }
{ "type": "publish", "room": "project:abc", "event": "file_updated", "data": { "path": "GRTC.tscn", "file_content": "base64..." } }
{ "type": "publish", "room": "project:abc", "event": "live_node", "data": { "path": "GRTC.tscn", "node_path": "Cube", "state": "T:0,2,0|R:0,0,0|S:1,1,1" } }
{ "type": "sync_request" }
```

**Server → Client** `welcome` • `auth_ok` • `room_joined` • `event` • `sync_state` • `pong` • `error`

**Presence**
```
GET /ws/online-users
GET /ws/user-status?user_id=<uuid>
```

Full reference: [`api.md`](./api.md)

## Project Structure

<p align="center"><img src="screenshots/structure.png" alt="Structure" width="600"></p>

```
rtc-server/
├── main.go              # env, DB init, router + CORS + rate limit
├── routers/routers.go   # /github/*, /health, /ws/*, /db/*, /api/*
├── services/            # github.go, middleware.go, websocket.go, activity.go ...
├── db/                  # postgres tables: users, projects, github_data ...
├── screenshots/
└── RTC demo.mp4         # ← this README's hero video
```

## Screenshots

<p align="center">
  <img src="screenshots/oauth.png" width="45%">
  <img src="screenshots/new_project.png" width="45%">
</p>
<p align="center">
  <img src="screenshots/logs.png" width="45%">
  <img src="screenshots/collaboration.png" width="45%">
</p>

## Tech Stack

`Go 1.24` • `net/http` + `gorilla/mux` + `gorilla/handlers` + `gorilla/websocket` • `PostgreSQL` + `lib/pq` • `golang.org/x/oauth2` • `google/uuid` • `joho/godotenv`

---

<p align="center">

**MIT** — github.com/Mohammad-416 • 2025-2026

Built for GRTC — Godot Realtime Collaboration

</p>
