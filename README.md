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

## Upcoming Features

Real-time collaboration with WebSockets

Room-based project sessions

Diff-based sync for scenes/scripts/assets

Better activity logging and history


**This Project is Licensed Under MIT License.**

**github.com/Mohammad-416** - 
**2025-2026**
