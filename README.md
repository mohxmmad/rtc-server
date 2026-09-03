# RTC Server

This backend now focuses on GitHub OAuth, sessions, user auth, and token/session restore.
Repo creation and collaborator management are deprecated.

The idea is to make it easy to spin up a project, invite people, and manage repos without doing everything manually.

## Features

Login with **GitHub OAuth** (session-based)

Session restore via saved session ID and backend token lookup

Basic repo actions like push/pull remain client-side for now

**User CRUD** (create, update, delete)

**Project CRUD** (create, update, delete)

**PostgreSQL** for storing users, projects, and sessions

Middleware for logging, CORS, etc. using ```gorilla/handlers + net/http```

## Tech Stack

**Language**: Go

**Auth**: GitHub OAuth

**Database**: PostgreSQL

**HTTP**: net/http + gorilla/handlers

**Platform**: GitHub OAuth + token/session storage

## Project Structure

<img src="screenshots/structure.png" alt="Structure" width="600">

## Examples


<img src="screenshots/oauth.png" alt="oauth" width="600">
<img src="screenshots/new_project.png" alt="new_project" width="600">
<img src="screenshots/logs.png" alt="logs" width="600">
<img src="screenshots/collaboration.png" alt="collaboration" width="600">

## Features - Real-time

*   **WebSocket live sync** - session-authenticated, room-based (`project:<id>`), auto-joins on connect
*   **File + scene sync** - file updates broadcast as `event` with base64 content, live transforms via `live_node`
*   **Unified protocol** for Unity and Godot: `auth` / `join` / `publish` / `sync_request` / `ping`

## WebSocket

Connect with a session ID:

```text
ws://localhost:8000/ws?session_id=<session_id>&project_id=<project_uuid>&client=unity
# or
ws://localhost:8000/ws?session_id=<session_id>&room=project:<hash>&client=godot
```

Protocol is JSON and shared by Unity and Godot clients:

```json
{ "type": "auth", "session_id": "...", "client": "unity" }
{ "type": "join", "project_id": "..." }
{ "type": "publish", "project_id": "...", "event": "file_updated", "data": { "path": "Assets/...", "file_content": "base64..." } }
{ "type": "publish", "room": "project:abc", "event": "live_node", "data": { "path": "GRTC.tscn", "node_path": "Cube", "state": "T:0,1,0|R:0,0,0|S:1,1,1" } }
{ "type": "sync_request" }
```

Server replies with `welcome`, `auth_ok`, `room_joined`, `event`, `sync_state`, `pong`, and `error`. Room is derived from normalized git remote URL so clones of the same repo share it.


**This Project is Licensed Under MIT License.**

**github.com/Mohammad-416** - 
**2025-2026**
