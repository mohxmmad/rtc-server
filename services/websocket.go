package services

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsMessageTypeWelcome   = "welcome"
	wsMessageTypeAuthOK    = "auth_ok"
	wsMessageTypeJoined    = "room_joined"
	wsMessageTypeLeft      = "room_left"
	wsMessageTypeEvent     = "event"
	wsMessageTypeError     = "error"
	wsMessageTypePong      = "pong"
	wsMessageTypeSyncState = "sync_state"
)

type WSIncoming struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	Room      string          `json:"room,omitempty"`
	ProjectID string          `json:"project_id,omitempty"`
	Event     string          `json:"event,omitempty"`
	Message   string          `json:"message,omitempty"`
	Client    string          `json:"client,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
}

type WSUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Client   string `json:"client,omitempty"`
}

type WSOut struct {
	Type        string                 `json:"type"`
	RequestID   string                 `json:"request_id,omitempty"`
	Room        string                 `json:"room,omitempty"`
	ProjectID   string                 `json:"project_id,omitempty"`
	Event       string                 `json:"event,omitempty"`
	Message     string                 `json:"message,omitempty"`
	Timestamp   string                 `json:"timestamp,omitempty"`
	User        *WSUser                `json:"user,omitempty"`
	OnlineUsers []string               `json:"online_users,omitempty"`
	TotalOnline int                    `json:"total_online,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

type wsClient struct {
	hub        *wsHub
	conn       *websocket.Conn
	send       chan []byte
	session    *Session
	clientName string
	rooms      map[string]struct{}
	registered bool
}

type wsHub struct {
	mu      sync.RWMutex
	clients map[*wsClient]struct{}
	rooms   map[string]map[*wsClient]struct{}
	users   map[string]map[*wsClient]struct{}
	locks   map[string]*roomLock
}

type roomLock struct {
	Locked           bool
	LockedByUserID   string
	LockedByUsername string
	Reason           string
	CreatedAt        time.Time
	PendingAcks      map[string]bool
	RequiredBytes    int64
	RequiredPath     string
}

var realtimeHub = newWSHub()

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" || origin == "null" {
			return true
		}

		allowed := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
		if allowed == "" || allowed == "*" {
			return true
		}

		for _, candidate := range strings.Split(allowed, ",") {
			candidate = strings.TrimSpace(candidate)
			if candidate == origin {
				return true
			}
			if candidate == "http://localhost" || candidate == "https://localhost" || candidate == "http://127.0.0.1" || candidate == "https://127.0.0.1" {
				if strings.HasPrefix(origin, candidate) {
					return true
				}
			}
			if candidate == "ws://localhost" || candidate == "wss://localhost" || candidate == "ws://127.0.0.1" || candidate == "wss://127.0.0.1" {
				origin = strings.Replace(origin, "http://", "ws://", 1)
				origin = strings.Replace(origin, "https://", "wss://", 1)
				if strings.HasPrefix(origin, candidate) {
					return true
				}
			}
		}

		log.Printf("websocket origin rejected: origin=%q allowed=%q", origin, allowed)
		return false
	},
}

func newWSHub() *wsHub {
	return &wsHub{
		clients: make(map[*wsClient]struct{}),
		rooms:   make(map[string]map[*wsClient]struct{}),
		users:   make(map[string]map[*wsClient]struct{}),
		locks:   make(map[string]*roomLock),
	}
}

func (h *wsHub) getRoomLock(room string) *roomLock {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.locks[normalizeRoom(room)]
}

func (h *wsHub) setRoomLock(room string, locker *wsClient, reason string, requiredPath string, requiredBytes int64) *roomLock {
	room = normalizeRoom(room)
	if room == "" {
		return nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	lock := &roomLock{
		Locked:        true,
		Reason:        reason,
		CreatedAt:     time.Now().UTC(),
		PendingAcks:   make(map[string]bool),
		RequiredBytes: requiredBytes,
		RequiredPath:  requiredPath,
	}
	if locker != nil && locker.session != nil {
		lock.LockedByUserID = locker.session.UserID.String()
		lock.LockedByUsername = locker.session.Username
	}
	if members, ok := h.rooms[room]; ok {
		for client := range members {
			if client.session != nil {
				lock.PendingAcks[client.session.UserID.String()] = true
			}
		}
	}
	h.locks[room] = lock
	return lock
}

func (h *wsHub) clearRoomLock(room string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.locks, normalizeRoom(room))
}

func (h *wsHub) markRoomAck(room string, userID string) (*roomLock, bool) {
	room = normalizeRoom(room)
	if room == "" || userID == "" {
		return nil, false
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	lock := h.locks[room]
	if lock == nil || !lock.Locked {
		return lock, false
	}
	delete(lock.PendingAcks, userID)
	if len(lock.PendingAcks) == 0 {
		lock.Locked = false
		delete(h.locks, room)
		return lock, true
	}
	return lock, false
}

func (h *wsHub) isRoomLocked(room string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	lock := h.locks[normalizeRoom(room)]
	return lock != nil && lock.Locked
}

func (h *wsHub) roomLockSnapshot(room string) map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()
	lock := h.locks[normalizeRoom(room)]
	if lock == nil {
		return map[string]interface{}{"locked": false}
	}
	pending := make([]string, 0, len(lock.PendingAcks))
	for userID := range lock.PendingAcks {
		pending = append(pending, userID)
	}
	sort.Strings(pending)
	return map[string]interface{}{
		"locked":             lock.Locked,
		"locked_by_user_id":  lock.LockedByUserID,
		"locked_by_username": lock.LockedByUsername,
		"reason":             lock.Reason,
		"created_at":         lock.CreatedAt.Format(time.RFC3339),
		"pending_acks":       pending,
		"required_path":      lock.RequiredPath,
		"required_bytes":     lock.RequiredBytes,
	}
}

func (h *wsHub) register(c *wsClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
	if c.session != nil {
		userID := c.session.UserID.String()
		if _, ok := h.users[userID]; !ok {
			h.users[userID] = make(map[*wsClient]struct{})
		}
		h.users[userID][c] = struct{}{}
	}
	c.registered = true
}

func (h *wsHub) unregister(c *wsClient) {
	roomsToUnlock := make([]string, 0)
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
	for room := range c.rooms {
		if members, ok := h.rooms[room]; ok {
			delete(members, c)
			if len(members) == 0 {
				delete(h.rooms, room)
			}
		}
		if lock := h.locks[room]; lock != nil && c.session != nil {
			delete(lock.PendingAcks, c.session.UserID.String())
			if len(lock.PendingAcks) == 0 {
				lock.Locked = false
				delete(h.locks, room)
				roomsToUnlock = append(roomsToUnlock, room)
			}
		}
	}
	if c.session != nil {
		userID := c.session.UserID.String()
		if members, ok := h.users[userID]; ok {
			delete(members, c)
			if len(members) == 0 {
				delete(h.users, userID)
			}
		}
	}
	for room := range c.rooms {
		delete(c.rooms, room)
	}
	c.registered = false
	if len(roomsToUnlock) > 0 {
		go func(rooms []string) {
			for _, room := range rooms {
				out := WSOut{Type: "sync_resumed", Room: room, Message: "room unlocked after client disconnect", Timestamp: time.Now().UTC().Format(time.RFC3339), Data: h.roomLockSnapshot(room)}
				payload, _ := json.Marshal(out)
				h.broadcast(room, payload, nil)
			}
		}(roomsToUnlock)
	}
}

func (h *wsHub) join(c *wsClient, room string) {
	room = normalizeRoom(room)
	if room == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.rooms[room]; !ok {
		h.rooms[room] = make(map[*wsClient]struct{})
	}
	h.rooms[room][c] = struct{}{}
	c.rooms[room] = struct{}{}
	if lock := h.locks[room]; lock != nil && lock.Locked && c.session != nil {
		lock.PendingAcks[c.session.UserID.String()] = true
	}
}

func (h *wsHub) leave(c *wsClient, room string) {
	room = normalizeRoom(room)
	if room == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if members, ok := h.rooms[room]; ok {
		delete(members, c)
		if len(members) == 0 {
			delete(h.rooms, room)
		}
	}
	delete(c.rooms, room)
}

func (h *wsHub) broadcast(room string, msg []byte, exclude *wsClient) int {
	room = normalizeRoom(room)
	if room == "" {
		return 0
	}

	h.mu.RLock()
	members := h.rooms[room]
	recipients := make([]*wsClient, 0, len(members))
	for c := range members {
		if c != exclude {
			recipients = append(recipients, c)
		}
	}
	h.mu.RUnlock()

	count := 0
	for _, c := range recipients {
		select {
		case c.send <- msg:
			count++
		default:
			go c.close()
		}
	}
	return count
}

func (h *wsHub) onlineUsers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	users := make([]string, 0, len(h.users))
	for userID := range h.users {
		users = append(users, userID)
	}
	sort.Strings(users)
	return users
}

func (h *wsHub) isUserOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.users[userID]
	return ok
}

func (h *wsHub) roomMembers(room string) []*wsClient {
	h.mu.RLock()
	defer h.mu.RUnlock()
	room = normalizeRoom(room)
	members := h.rooms[room]
	clients := make([]*wsClient, 0, len(members))
	for c := range members {
		clients = append(clients, c)
	}
	return clients
}

func normalizeRoom(room string) string {
	room = strings.TrimSpace(room)
	if room == "" {
		return ""
	}
	if strings.HasPrefix(room, "project:") {
		return room
	}
	if len(room) == 36 {
		return "project:" + room
	}
	return room
}

func roomFromProjectID(projectID string) string {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return ""
	}
	return "project:" + projectID
}

func strOrDefault(value interface{}, fallback string) string {
	if value == nil {
		return fallback
	}
	if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
		return s
	}
	return fallback
}

func int64FromAny(value interface{}) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		parsed, err := v.Int64()
		if err == nil {
			return parsed
		}
	}
	return 0
}

func (c *wsClient) close() {
	_ = c.conn.Close()
}

func (c *wsClient) sendJSON(message any) {
	payload, err := json.Marshal(message)
	if err != nil {
		return
	}
	select {
	case c.send <- payload:
	default:
		go c.close()
	}
}

func (c *wsClient) authFromSessionID(sessionID, clientName string) bool {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return false
	}
	if session, ok := sessionManager.GetSession(sessionID); ok {
		c.session = session
		if clientName != "" {
			c.clientName = clientName
		}
		if !c.registered {
			c.hub.register(c)
		}
		return true
	}
	return false
}

func (c *wsClient) authFromIncoming(msg WSIncoming) bool {
	return c.authFromSessionID(msg.SessionID, msg.Client)
}

func (c *wsClient) handleMessage(msg WSIncoming) {
	switch strings.ToLower(strings.TrimSpace(msg.Type)) {
	case "auth":
		if c.authFromIncoming(msg) {
			c.sendJSON(WSOut{
				Type:      wsMessageTypeAuthOK,
				Message:   "authenticated",
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				User: &WSUser{
					ID:       c.session.UserID.String(),
					Username: c.session.Username,
					Email:    c.session.Email,
					Client:   c.clientName,
				},
			})
			if room := normalizeRoom(msg.Room); room != "" {
				c.hub.join(c, room)
				c.sendJSON(WSOut{Type: wsMessageTypeJoined, Room: room, Timestamp: time.Now().UTC().Format(time.RFC3339), Data: c.hub.roomLockSnapshot(room)})
			}
			if projectRoom := roomFromProjectID(msg.ProjectID); projectRoom != "" {
				c.hub.join(c, projectRoom)
				c.sendJSON(WSOut{Type: wsMessageTypeJoined, Room: projectRoom, ProjectID: msg.ProjectID, Timestamp: time.Now().UTC().Format(time.RFC3339), Data: c.hub.roomLockSnapshot(projectRoom)})
			}
			return
		}
		c.sendJSON(WSOut{Type: wsMessageTypeError, Message: "invalid or expired session_id", Timestamp: time.Now().UTC().Format(time.RFC3339)})
	case "join":
		if !c.ensureAuthed() {
			return
		}
		room := msg.Room
		if room == "" {
			room = roomFromProjectID(msg.ProjectID)
		}
		room = normalizeRoom(room)
		if room == "" {
			c.sendJSON(WSOut{Type: wsMessageTypeError, Message: "room or project_id is required", RequestID: msg.RequestID, Timestamp: time.Now().UTC().Format(time.RFC3339)})
			return
		}
		c.hub.join(c, room)
		c.sendJSON(WSOut{Type: wsMessageTypeJoined, RequestID: msg.RequestID, Room: room, ProjectID: msg.ProjectID, Timestamp: time.Now().UTC().Format(time.RFC3339), Data: c.hub.roomLockSnapshot(room)})
	case "leave":
		if !c.ensureAuthed() {
			return
		}
		room := normalizeRoom(msg.Room)
		if room == "" {
			room = roomFromProjectID(msg.ProjectID)
		}
		if room == "" {
			c.sendJSON(WSOut{Type: wsMessageTypeError, Message: "room or project_id is required", RequestID: msg.RequestID, Timestamp: time.Now().UTC().Format(time.RFC3339)})
			return
		}
		c.hub.leave(c, room)
		c.sendJSON(WSOut{Type: wsMessageTypeLeft, RequestID: msg.RequestID, Room: room, ProjectID: msg.ProjectID, Timestamp: time.Now().UTC().Format(time.RFC3339), Data: c.hub.roomLockSnapshot(room)})
	case "publish":
		if !c.ensureAuthed() {
			return
		}
		room := normalizeRoom(msg.Room)
		if room == "" {
			room = roomFromProjectID(msg.ProjectID)
		}
		if room == "" {
			c.sendJSON(WSOut{Type: wsMessageTypeError, Message: "room or project_id is required", RequestID: msg.RequestID, Timestamp: time.Now().UTC().Format(time.RFC3339)})
			return
		}
		event := strings.TrimSpace(msg.Event)
		if event == "" {
			event = "message"
		}
		data := map[string]interface{}{}
		if len(msg.Data) > 0 {
			_ = json.Unmarshal(msg.Data, &data)
		}
		out := WSOut{
			Type:      wsMessageTypeEvent,
			RequestID: msg.RequestID,
			Room:      room,
			ProjectID: msg.ProjectID,
			Event:     event,
			Message:   msg.Message,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			User: &WSUser{
				ID:       c.session.UserID.String(),
				Username: c.session.Username,
				Email:    c.session.Email,
				Client:   c.clientName,
			},
			Data: data,
		}
		payload, err := json.Marshal(out)
		if err != nil {
			c.sendJSON(WSOut{Type: wsMessageTypeError, Message: "failed to encode event", RequestID: msg.RequestID, Timestamp: time.Now().UTC().Format(time.RFC3339)})
			return
		}
		c.hub.broadcast(room, payload, nil)
	case "push_required":
		if !c.ensureAuthed() {
			return
		}
		room := normalizeRoom(msg.Room)
		if room == "" {
			room = roomFromProjectID(msg.ProjectID)
		}
		if room == "" {
			c.sendJSON(WSOut{Type: wsMessageTypeError, Message: "room or project_id is required", RequestID: msg.RequestID, Timestamp: time.Now().UTC().Format(time.RFC3339)})
			return
		}
		data := map[string]interface{}{}
		if len(msg.Data) > 0 {
			_ = json.Unmarshal(msg.Data, &data)
		}
		lock := c.hub.setRoomLock(room, c, strOrDefault(data["reason"], "large change requires push"), strOrDefault(data["required_path"], ""), int64FromAny(data["required_bytes"]))
		out := WSOut{Type: "sync_paused", Room: room, ProjectID: msg.ProjectID, Event: "push_required", Message: "live sync paused until push and pull complete", Timestamp: time.Now().UTC().Format(time.RFC3339), Data: map[string]interface{}{"lock": lock}}
		payload, _ := json.Marshal(out)
		c.hub.broadcast(room, payload, nil)
	case "push_complete":
		if !c.ensureAuthed() {
			return
		}
		room := normalizeRoom(msg.Room)
		if room == "" {
			room = roomFromProjectID(msg.ProjectID)
		}
		if room == "" {
			c.sendJSON(WSOut{Type: wsMessageTypeError, Message: "room or project_id is required", RequestID: msg.RequestID, Timestamp: time.Now().UTC().Format(time.RFC3339)})
			return
		}
		out := WSOut{Type: "push_complete", Room: room, ProjectID: msg.ProjectID, Message: "push completed; waiting for everyone to pull", Timestamp: time.Now().UTC().Format(time.RFC3339), Data: c.hub.roomLockSnapshot(room)}
		payload, _ := json.Marshal(out)
		c.hub.broadcast(room, payload, nil)
	case "client_synced", "room_synced", "pull_complete":
		if !c.ensureAuthed() {
			return
		}
		room := normalizeRoom(msg.Room)
		if room == "" {
			room = roomFromProjectID(msg.ProjectID)
		}
		if room == "" {
			c.sendJSON(WSOut{Type: wsMessageTypeError, Message: "room or project_id is required", RequestID: msg.RequestID, Timestamp: time.Now().UTC().Format(time.RFC3339)})
			return
		}
		lock, cleared := c.hub.markRoomAck(room, c.session.UserID.String())
		if lock == nil {
			c.sendJSON(WSOut{Type: wsMessageTypeSyncState, RequestID: msg.RequestID, Room: room, Timestamp: time.Now().UTC().Format(time.RFC3339), Data: map[string]interface{}{"locked": false}})
			return
		}
		if cleared {
			out := WSOut{Type: "sync_resumed", Room: room, ProjectID: msg.ProjectID, Message: "room unlocked; live sync resumed", Timestamp: time.Now().UTC().Format(time.RFC3339), Data: c.hub.roomLockSnapshot(room)}
			payload, _ := json.Marshal(out)
			c.hub.broadcast(room, payload, nil)
			return
		}
		c.sendJSON(WSOut{Type: wsMessageTypeSyncState, RequestID: msg.RequestID, Room: room, Timestamp: time.Now().UTC().Format(time.RFC3339), Data: c.hub.roomLockSnapshot(room)})
	case "ping":
		c.sendJSON(WSOut{Type: wsMessageTypePong, RequestID: msg.RequestID, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	case "whoami":
		if !c.ensureAuthed() {
			return
		}
		c.sendJSON(WSOut{
			Type:      wsMessageTypeAuthOK,
			RequestID: msg.RequestID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			User: &WSUser{
				ID:       c.session.UserID.String(),
				Username: c.session.Username,
				Email:    c.session.Email,
				Client:   c.clientName,
			},
		})
	case "sync_request":
		if !c.ensureAuthed() {
			return
		}
		rooms := make([]string, 0, len(c.rooms))
		for room := range c.rooms {
			rooms = append(rooms, room)
		}
		sort.Strings(rooms)
		onlineUsers := realtimeHub.onlineUsers()
		roomLock := map[string]interface{}{"locked": false}
		if len(rooms) > 0 {
			roomLock = c.hub.roomLockSnapshot(rooms[0])
		}
		c.sendJSON(WSOut{
			Type:        wsMessageTypeSyncState,
			RequestID:   msg.RequestID,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
			OnlineUsers: onlineUsers,
			TotalOnline: len(onlineUsers),
			Data: map[string]interface{}{
				"joined_rooms": rooms,
				"client":       c.clientName,
				"room_lock":    roomLock,
			},
		})
	default:
		c.sendJSON(WSOut{Type: wsMessageTypeError, Message: "unsupported message type", RequestID: msg.RequestID, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	}
}

func (c *wsClient) ensureAuthed() bool {
	if c.session != nil {
		return true
	}
	c.sendJSON(WSOut{Type: wsMessageTypeError, Message: "auth required", Timestamp: time.Now().UTC().Format(time.RFC3339)})
	return false
}

func (c *wsClient) readPump() {
	defer func() {
		if c.registered {
			c.hub.unregister(c)
		}
		c.close()
	}()

	c.conn.SetReadLimit(10 * 1024 * 1024)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	if c.session != nil {
		c.sendJSON(WSOut{
			Type:      wsMessageTypeWelcome,
			Message:   "connected to realtime collaboration server",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			User: &WSUser{
				ID:       c.session.UserID.String(),
				Username: c.session.Username,
				Email:    c.session.Email,
				Client:   c.clientName,
			},
		})
	} else {
		c.sendJSON(WSOut{
			Type:      wsMessageTypeWelcome,
			Message:   "send an auth message with session_id",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	}

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var incoming WSIncoming
		if err := json.Unmarshal(message, &incoming); err != nil {
			c.sendJSON(WSOut{Type: wsMessageTypeError, Message: "invalid json payload", Timestamp: time.Now().UTC().Format(time.RFC3339)})
			continue
		}

		if c.session == nil && strings.ToLower(strings.TrimSpace(incoming.Type)) != "auth" && strings.ToLower(strings.TrimSpace(incoming.Type)) != "ping" {
			c.sendJSON(WSOut{Type: wsMessageTypeError, Message: "auth required", RequestID: incoming.RequestID, Timestamp: time.Now().UTC().Format(time.RFC3339)})
			continue
		}

		c.handleMessage(incoming)
	}
}

func (c *wsClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
				return
			}
		}
	}
}

// WebSocketHandler upgrades the connection and attaches the client to the realtime hub.
func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &wsClient{
		hub:   realtimeHub,
		conn:  conn,
		send:  make(chan []byte, 32),
		rooms: make(map[string]struct{}),
	}

	if client.authFromSessionID(r.URL.Query().Get("session_id"), r.URL.Query().Get("client")) {
		if projectRoom := roomFromProjectID(r.URL.Query().Get("project_id")); projectRoom != "" {
			client.hub.join(client, projectRoom)
		}
		if room := normalizeRoom(r.URL.Query().Get("room")); room != "" {
			client.hub.join(client, room)
		}
	}

	go client.writePump()
	client.readPump()
}

// OnlineUsersHandler returns the current set of online user IDs.
func OnlineUsersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	onlineUsers := realtimeHub.onlineUsers()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"online_users": onlineUsers,
		"total_online": len(onlineUsers),
	})
}

// UserStatusHandler returns whether a user is online.
func UserStatusHandler(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if userID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "user_id is required"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"user_id":   userID,
		"is_online": realtimeHub.isUserOnline(userID),
	})
}

// BroadcastActivityToProject sends an activity event to the project room.
func BroadcastActivityToProject(projectID string, payload map[string]interface{}) {
	room := roomFromProjectID(projectID)
	if room == "" {
		return
	}
	msg := WSOut{
		Type:      wsMessageTypeEvent,
		Room:      room,
		ProjectID: projectID,
		Event:     "activity",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      payload,
	}
	encoded, err := json.Marshal(msg)
	if err != nil {
		log.Printf("websocket broadcast encode error: %v", err)
		return
	}
	realtimeHub.broadcast(room, encoded, nil)
}
