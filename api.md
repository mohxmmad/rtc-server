# RTC Server API Documentation

## 🔐 Authentication

Most endpoints require authentication via GitHub OAuth token. Include the token in the `Authorization` header:

```
Authorization: Bearer <github_token>
```

Or use the `X-User-ID` header for WebSocket and real-time features:

```
X-User-ID: <user_uuid>
```

---

## 👥 User Endpoints

### Get User by Username
```http
GET /db/users/{username}
```

### Get User by Email
```http
GET /db/user/{email}
```

### Get All Users
```http
GET /db/users
```

### Get Users Count
```http
GET /db/users-count
```

### Delete User
```http
DELETE /db/users/{username}
```

---

## 📦 Project Endpoints

### Get Projects Count
```http
GET /db/projects-count/{owner}
```

### Get All Projects for Owner
```http
GET /db/projects/{owner}
```

### Get Specific Project
```http
GET /db/projects/{owner}/{project_name}
```

### Delete Project
```http
DELETE /db/projects/{owner}/{project_name}
```

### Push Project (Manual)
```http
POST /push/manual
Content-Type: application/json

{
  "owner": "owner@abc.com",
  "project_name": "MyUnityGame"
}
```

---

## 🤝 Collaboration Endpoints

### Request Collaboration
```http
POST /collab/request
Content-Type: application/json

{
  "owner_email": "owner@example.com",
  "collaborator_email": "collab@example.com",
  "project_id": "uuid-here"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Collaboration request sent successfully",
  "collab_id": "uuid",
  "status": "pending",
  "notification": {
    "collab_id": "uuid",
    "project_name": "MyProject",
    "project_owner": "owner"
  }
}
```

### Approve/Reject Collaboration
```http
POST /collab/approve
Content-Type: application/json

{
  "collab_id": "uuid-here",
  "status": "approved",  // or "rejected"
  "approver_token": "github_token_here"
}
```

### Get Project Collaborators
```http
GET /collab/project?project_id={project_uuid}
```

**Response:**
```json
{
  "success": true,
  "project_id": "uuid",
  "collaborators": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "project_id": "uuid",
      "status": "approved",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

### Get User Collaboration Requests
```http
GET /collab/user/requests?user_email={email}
```

### Remove Collaborator
```http
DELETE /collab/remove/{collab_id}
```

---

## 📁 File Sharing Endpoints

### Share Single File
```http
POST /share/file
Content-Type: application/json

{
  "sender_email": "sender@example.com",
  "recipient_email": "recipient@example.com",
  "project_id": "uuid",
  "file_name": "PlayerController.cs",
  "file_content": "base64_encoded_content",
  "file_type": "script",
  "message": "Check out this new controller!"
}
```

### Share Code Snippet
```http
POST /share/code
Content-Type: application/json

{
  "sender_email": "sender@example.com",
  "recipient_email": "recipient@example.com",
  "project_id": "uuid",
  "file_name": "GameManager.cs",
  "code": "public void StartGame() { ... }",
  "language": "csharp",
  "line_number": 42,
  "message": "Look at this function"
}
```

### Share Multiple Files (Bulk)
```http
POST /share/bulk
Content-Type: application/json

{
  "sender_email": "sender@example.com",
  "recipient_email": "recipient@example.com",
  "project_id": "uuid",
  "files": [
    {
      "file_name": "Sprite1.png",
      "file_content": "base64_content",
      "file_type": "asset"
    },
    {
      "file_name": "Sprite2.png",
      "file_content": "base64_content",
      "file_type": "asset"
    }
  ],
  "message": "Here are the new sprites"
}
```

### Get Shareable Collaborators
```http
GET /share/collaborators?project_id={project_uuid}
```

**Response:**
```json
{
  "success": true,
  "project_id": "uuid",
  "collaborators": [
    {
      "user_id": "uuid",
      "username": "developer1",
      "email": "dev@example.com",
      "status": "approved",
      "is_online": true
    }
  ],
  "total": 1
}
```

---

## 📊 Activity Tracking Endpoints

### Get User Activities
```http
GET /activity/user?user_email={email}&limit=50
```

**Response:**
```json
{
  "success": true,
  "user_email": "user@example.com",
  "activities": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "project_id": "uuid",
      "action": "file_commit",
      "description": "Committed PlayerController.cs",
      "metadata": {
        "file_path": "Assets/Scripts/PlayerController.cs",
        "version": 5
      },
      "ip_address": "192.168.1.1",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

### Get Project Activities
```http
GET /activity/project?project_id={uuid}&limit=50
```

### Get Team Activities
```http
GET /activity/team?project_id={uuid}&limit=100
```

---

## 📝 Version Control Endpoints

### Commit File Version
```http
POST /version/commit
Content-Type: application/json

{
  "project_id": "uuid",
  "user_email": "user@example.com",
  "file_path": "Assets/Scripts/GameManager.cs",
  "file_name": "GameManager.cs",
  "file_type": "script",
  "content": "file_content_here",
  "file_hash": "sha256_hash",
  "file_size": 1024,
  "commit_message": "Added game state management",
  "base_version": 4
}
```

**Response (Success):**
```json
{
  "success": true,
  "version_id": "uuid",
  "version": 5,
  "file_path": "Assets/Scripts/GameManager.cs",
  "commit_msg": "Added game state management",
  "has_conflict": false
}
```

**Response (Conflict Detected):**
```json
{
  "success": false,
  "conflict": true,
  "conflict_id": "uuid",
  "message": "Conflict detected. Please resolve before committing.",
  "base_version": 4,
  "latest_version": 5
}
```

### Get File History
```http
GET /version/history?project_id={uuid}&file_path={path}
```

**Response:**
```json
{
  "success": true,
  "project_id": "uuid",
  "file_path": "Assets/Scripts/GameManager.cs",
  "versions": [
    {
      "id": "uuid",
      "version": 5,
      "user_id": "uuid",
      "commit_message": "Added game state management",
      "file_hash": "sha256",
      "file_size": 1024,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 5
}
```

### Get Project Versions
```http
GET /version/project?project_id={uuid}
```

### Get File Conflicts
```http
GET /version/conflicts?project_id={uuid}
```

**Response:**
```json
{
  "success": true,
  "project_id": "uuid",
  "conflicts": [
    {
      "id": "uuid",
      "project_id": "uuid",
      "file_path": "Assets/Scripts/PlayerController.cs",
      "base_version": 3,
      "local_user_id": "uuid1",
      "remote_user_id": "uuid2",
      "status": "pending",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

### Resolve Conflict
```http
POST /version/resolve
Content-Type: application/json

{
  "conflict_id": "uuid",
  "resolved_by_email": "user@example.com",
  "resolved_content": "merged_content_here",
  "commit_message": "Resolved merge conflict"
}
```

---

## 🔌 WebSocket Endpoints

### Connect to WebSocket
```
ws://localhost:8080/ws?user_id={user_uuid}
```

### WebSocket Message Types

#### Connection Success
```json
{
  "type": "connection_success",
  "message": "Connected to real-time collaboration server",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

#### File Share Notification
```json
{
  "type": "file_share",
  "sender_id": "uuid",
  "sender_email": "sender@example.com",
  "recipient_id": "uuid",
  "file_name": "PlayerController.cs",
  "file_content": "base64_content",
  "file_type": "script",
  "message": "Check this out!",
  "timestamp": "2024-01-01T00:00:00Z",
  "metadata": {
    "project_id": "uuid"
  }
}
```

#### Code Share Notification
```json
{
  "type": "code_share",
  "sender_id": "uuid",
  "code": "public void Update() { ... }",
  "language": "csharp",
  "message": "New update function",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

#### File Updated Notification
```json
{
  "type": "file_updated",
  "message": "developer1 committed PlayerController.cs",
  "timestamp": "2024-01-01T00:00:00Z",
  "metadata": {
    "project_id": "uuid",
    "file_path": "Assets/Scripts/PlayerController.cs",
    "version": 5,
    "user_email": "dev@example.com"
  }
}
```

#### Collaboration Request Notification
```json
{
  "type": "collaboration_request",
  "message": "New collaboration request",
  "timestamp": "2024-01-01T00:00:00Z",
  "metadata": {
    "collab_id": "uuid",
    "project_id": "uuid",
    "project_name": "MyGame",
    "project_owner": "owner_username"
  }
}
```

#### Conflict Notification
```json
{
  "type": "file_conflict",
  "message": "File conflict detected",
  "timestamp": "2024-01-01T00:00:00Z",
  "metadata": {
    "conflict_id": "uuid",
    "project_id": "uuid",
    "file_path": "Assets/Scripts/GameManager.cs",
    "user_email": "user@example.com"
  }
}
```

### Get Online Users
```http
GET /ws/online-users
```

**Response:**
```json
{
  "success": true,
  "online_users": ["uuid1", "uuid2", "uuid3"],
  "total_online": 3
}
```

### Check User Online Status
```http
GET /ws/user-status?user_id={uuid}
```

**Response:**
```json
{
  "success": true,
  "user_id": "uuid",
  "is_online": true
}
```

---

## 🔑 GitHub OAuth Endpoints

### Initiate Login
```http
GET /github/login
```
Redirects to GitHub OAuth page.

### OAuth Callback
```http
GET /github/callback?code={auth_code}
```
Handles OAuth callback and stores token.

### Get GitHub Token
```http
GET /db/token/{super_user_key}/{username}
```
Protected endpoint to retrieve GitHub token.

---

## 🏥 Health Check

### Server Health
```http
GET /health
```

**Response:**
```
OK
```

---

## 📋 Common Status Codes

- `200 OK` - Request successful
- `201 Created` - Resource created successfully
- `400 Bad Request` - Invalid request parameters
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Access denied
- `404 Not Found` - Resource not found
- `409 Conflict` - Conflict detected (version control)
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error

---

## 🛡️ Rate Limiting

The server implements rate limiting:
- **60 requests per minute** per IP address
- **Burst of 10 requests** allowed

When rate limit is exceeded:
```json
{
  "error": "Rate limit exceeded. Please try again later."
}
```

---

## 🔒 Security Headers

All responses include security headers:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Strict-Transport-Security: max-age=31536000`

---

## 📦 Request/Response Format

All API requests and responses use JSON format:

```http
Content-Type: application/json
```

Standard error response:
```json
{
  "error": "Error message here"
}
```

Standard success response:
```json
{
  "success": true,
  "message": "Operation successful",
  "data": { ... }
}
```
