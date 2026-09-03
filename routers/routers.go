package routers

import (
	"app/rtc/services"
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
	r.HandleFunc("/github/latest-auth", services.GetLatestAuth).Methods("GET")
	r.Handle("/github/session-auth", services.SessionAuth(http.HandlerFunc(services.GetSessionAuth))).Methods("GET")
	r.HandleFunc("/github/logout", services.LogoutHandler).Methods("POST")

	// Health check (No auth required)
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// WebSocket realtime endpoints
	r.HandleFunc("/ws", services.WebSocketHandler).Methods("GET")
	r.HandleFunc("/ws/online-users", services.OnlineUsersHandler).Methods("GET")
	r.HandleFunc("/ws/user-status", services.UserStatusHandler).Methods("GET")

	// PUBLIC ROUTES - No session required
	publicRoutes := r.PathPrefix("").Subrouter()
	publicRoutes.Use(services.CORS)
	publicRoutes.Use(services.RequestLogger)
	publicRoutes.Use(services.Recovery)
	publicRoutes.Use(rateLimiter.Limit)

	// User Functions (Public - authentication support)
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

	// Activity Tracking Routes (Protected)
	protectedRoutes.HandleFunc("/activity/user", services.GetUserActivities).Methods("GET")
	protectedRoutes.HandleFunc("/activity/project", services.GetProjectActivities).Methods("GET")
	protectedRoutes.HandleFunc("/activity/team", services.GetRecentTeamActivities).Methods("GET")

	// User Management (Protected)
	protectedRoutes.HandleFunc("/users/{user}", services.DeleteUser).Methods("DELETE")

	// ADMIN ROUTES - Super user key required
	adminRoutes := r.PathPrefix("/admin").Subrouter()
	adminRoutes.Use(services.CORS)
	adminRoutes.Use(services.RequestLogger)
	adminRoutes.Use(services.Recovery)

	// GitHub Token Access (Admin only)
	adminRoutes.HandleFunc("/token/{super_user_key}/{user}", services.GetToken).Methods("GET")

	return r
}
