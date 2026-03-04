package routers

import (
	"app/urtc/services"
	"net/http"

	"github.com/gorilla/mux"
)

func SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Create rate limiter
	rateLimiter := services.NewRateLimiter(60, 10) // 60 requests per minute, burst of 10

	// GitHub OAuth (No auth required)
	r.HandleFunc("/github/login", services.GitHubLoginHandler).Methods("GET")
	r.HandleFunc("/github/callback", services.GitHubCallbackHandler).Methods("GET")
	r.HandleFunc("/github/logout", services.LogoutHandler).Methods("POST")

	// Health check (No auth required)
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// PUBLIC ROUTES - No session required
	publicRoutes := r.PathPrefix("").Subrouter()
	publicRoutes.Use(services.CORS)
	publicRoutes.Use(services.RequestLogger)
	publicRoutes.Use(services.Recovery)
	publicRoutes.Use(rateLimiter.Limit)

	// User Functions (Public - for backward compatibility)
	publicRoutes.HandleFunc("/db/users-count", services.GetUsersLen).Methods("GET")
	publicRoutes.HandleFunc("/db/users", services.GetUsers).Methods("GET")
	publicRoutes.HandleFunc("/db/users/{user}", services.GetUser).Methods("GET")
	publicRoutes.HandleFunc("/db/user/{email}", services.GetUserByEmail).Methods("GET")

	// PROTECTED ROUTES - Session authentication required
	protectedRoutes := r.PathPrefix("/api").Subrouter()
	protectedRoutes.Use(services.CORS)
	protectedRoutes.Use(services.RequestLogger)
	protectedRoutes.Use(services.Recovery)
	protectedRoutes.Use(services.SessionAuth) // SESSION AUTH MIDDLEWARE
	protectedRoutes.Use(rateLimiter.Limit)

	// Project Functions (Protected)
	protectedRoutes.HandleFunc("/projects-count/{owner}", services.NProjects).Methods("GET")
	protectedRoutes.HandleFunc("/projects/{owner}", services.GetProjects).Methods("GET")
	protectedRoutes.HandleFunc("/projects/{owner}/{name}", services.GetProject).Methods("GET")
	protectedRoutes.HandleFunc("/projects/{owner}/{name}", services.DeleteProject).Methods("DELETE")

	// Push Project (Protected)
	protectedRoutes.HandleFunc("/push/manual", services.PushProject).Methods("POST")

	// Collaborator Routes (Protected with Session Auth)
	protectedRoutes.HandleFunc("/collab/request", services.RequestCollaboration).Methods("POST")
	protectedRoutes.HandleFunc("/collab/approve", services.ApproveCollaboration).Methods("POST")
	protectedRoutes.HandleFunc("/collab/project", services.GetProjectCollaborators).Methods("GET")
	protectedRoutes.HandleFunc("/collab/user/requests", services.GetUserCollaborationRequests).Methods("GET")
	protectedRoutes.HandleFunc("/collab/remove/{collab_id}", services.RemoveCollaborator).Methods("DELETE")

	// NEW ENDPOINT: Get pending collaboration requests for owner's project
	protectedRoutes.HandleFunc("/collab/pending", services.GetProjectPendingCollaborations).Methods("GET")

	// WebSocket Routes (Protected)
	protectedRoutes.HandleFunc("/ws", services.HandleWebSocket).Methods("GET")
	protectedRoutes.HandleFunc("/ws/online-users", services.GetOnlineUsers).Methods("GET")
	protectedRoutes.HandleFunc("/ws/user-status", services.CheckUserOnlineStatus).Methods("GET")

	// File Sharing Routes (Protected)
	protectedRoutes.HandleFunc("/share/file", services.ShareFile).Methods("POST")
	protectedRoutes.HandleFunc("/share/code", services.ShareCode).Methods("POST")
	protectedRoutes.HandleFunc("/share/bulk", services.ShareBulkFiles).Methods("POST")
	protectedRoutes.HandleFunc("/share/collaborators", services.GetShareableCollaborators).Methods("GET")

	// Activity Tracking Routes (Protected)
	protectedRoutes.HandleFunc("/activity/user", services.GetUserActivities).Methods("GET")
	protectedRoutes.HandleFunc("/activity/project", services.GetProjectActivities).Methods("GET")
	protectedRoutes.HandleFunc("/activity/team", services.GetRecentTeamActivities).Methods("GET")

	// Version Control Routes (Protected)
	protectedRoutes.HandleFunc("/version/commit", services.CommitFileVersion).Methods("POST")
	protectedRoutes.HandleFunc("/version/history", services.GetFileHistory).Methods("GET")
	protectedRoutes.HandleFunc("/version/project", services.GetProjectVersions).Methods("GET")
	protectedRoutes.HandleFunc("/version/conflicts", services.GetFileConflicts).Methods("GET")
	protectedRoutes.HandleFunc("/version/resolve", services.ResolveConflict).Methods("POST")

	// User Management (Protected)
	protectedRoutes.HandleFunc("/users/{user}", services.DeleteUser).Methods("DELETE")

	// Unity Plugin Compatibility Routes
	protectedRoutes.HandleFunc("/start-collaboration", services.PushProject).Methods("POST")
	protectedRoutes.HandleFunc("/join-collaboration", services.JoinCollaboration).Methods("POST")

	// ADMIN ROUTES - Super user key required
	adminRoutes := r.PathPrefix("/admin").Subrouter()
	adminRoutes.Use(services.CORS)
	adminRoutes.Use(services.RequestLogger)
	adminRoutes.Use(services.Recovery)

	// GitHub Token Access (Admin only)
	adminRoutes.HandleFunc("/token/{super_user_key}/{user}", services.GetToken).Methods("GET")

	return r
}
