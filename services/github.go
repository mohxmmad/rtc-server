package services

import (
	"app/rtc/db"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var githubOAuthConfig *oauth2.Config

type LatestAuth struct {
	SessionID   string    `json:"session_id"`
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	GitHubToken string    `json:"github_token"`
	CreatedAt   time.Time `json:"created_at"`
}

var latestAuthMu sync.RWMutex
var latestAuthByState = map[string]*LatestAuth{}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	githubOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		Scopes:       []string{"repo", "user"},
		Endpoint:     github.Endpoint,
		RedirectURL:  os.Getenv("GITHUB_CALLBACK_URL"),
	}
}

func GitHubLoginHandler(w http.ResponseWriter, r *http.Request) {
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if state == "" {
		state = "randomstate"
	}
	url := githubOAuthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusFound)
}

func GitHubCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	token, err := githubOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	accessToken := token.AccessToken

	client := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(token))
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var user struct {
		Login string `json:"login"`
		ID    int    `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		http.Error(w, "Failed to parse user info", http.StatusInternalServerError)
		return
	}

	// If email is empty, fetch /user/emails
	if user.Email == "" {
		emailResp, err := client.Get("https://api.github.com/user/emails")
		if err != nil {
			http.Error(w, "Failed to get user emails", http.StatusInternalServerError)
			return
		}
		defer emailResp.Body.Close()

		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}

		if err := json.NewDecoder(emailResp.Body).Decode(&emails); err != nil {
			http.Error(w, "Failed to parse user emails", http.StatusInternalServerError)
			return
		}

		for _, e := range emails {
			if e.Primary && e.Verified {
				user.Email = e.Email
				break
			}
		}
	}

	if user.Email == "" {
		http.Error(w, "Email not available, please make your email public on GitHub", http.StatusBadRequest)
		return
	}

	userModel := &db.UserModel{DB: db.DB}

	var newUser *db.User
	// Check if the user already exists
	existingUser, err := userModel.GetUserByEmail(user.Email)
	if err == nil {
		// User exists, update token
		newUser = existingUser
		stored_data := StoreAccessToken(newUser.USERNAME, accessToken, newUser.ID)
		if stored_data {
			fmt.Println("Data Updated Successfully")
		} else {
			fmt.Println("Unable to update token")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Printf("Welcome back, %s! Your email is %s\n", newUser.USERNAME, newUser.EMAIL)
	} else {
		// Create new user
		createdUser, err := userModel.CreateUser(int64(user.ID), user.Login, user.Email)
		if err != nil {
			http.Error(w, "Failed to save user: "+err.Error(), http.StatusInternalServerError)
			return
		}
		newUser = createdUser

		// Store the github data
		stored_data := StoreAccessToken(newUser.USERNAME, accessToken, newUser.ID)
		if stored_data {
			fmt.Println("Data Stored Successfully")
		} else {
			fmt.Println("Unable to save token")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Printf("Welcome, %s! Your email is %s\n", newUser.USERNAME, newUser.EMAIL)
	}

	// CREATE SESSION FOR THE USER
	sessionID, err := sessionManager.CreateSession(newUser.ID, newUser.USERNAME, newUser.EMAIL)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if state != "" {
		saveLatestAuth(state, &LatestAuth{
			SessionID:   sessionID,
			UserID:      newUser.ID.String(),
			Username:    newUser.USERNAME,
			Email:       newUser.EMAIL,
			GitHubToken: accessToken,
			CreatedAt:   time.Now(),
		})
	}

	// Return session ID in response header and body
	w.Header().Set("X-Session-ID", sessionID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"message":    "Login successful",
		"session_id": sessionID,
		"user": map[string]interface{}{
			"id":       newUser.ID,
			"username": newUser.USERNAME,
			"email":    newUser.EMAIL,
		},
	})

	fmt.Println("\n✅ Session created:", sessionID)
	fmt.Println("You can now use this session_id for authenticated requests")
}

func saveLatestAuth(state string, auth *LatestAuth) {
	state = strings.TrimSpace(state)
	if state == "" {
		return
	}
	latestAuthMu.Lock()
	defer latestAuthMu.Unlock()
	latestAuthByState[state] = auth
}

func GetLatestAuth(w http.ResponseWriter, r *http.Request) {
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if state == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "state is required"})
		return
	}

	latestAuthMu.RLock()
	auth, ok := latestAuthByState[state]
	latestAuthMu.RUnlock()
	if !ok || auth == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "login snapshot not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"session_id":   auth.SessionID,
		"github_token": auth.GitHubToken,
		"user": map[string]interface{}{
			"id":       auth.UserID,
			"username": auth.Username,
			"email":    auth.Email,
		},
		"created_at": auth.CreatedAt,
	})
}

func GetSessionAuth(w http.ResponseWriter, r *http.Request) {
	session, ok := GetSessionFromContext(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Session not found"})
		return
	}

	tokenModel := &db.TokenModel{DB: db.DB}
	token, err := tokenModel.GetToken(session.Username)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "GitHub token not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"session_id":   session.ID,
		"github_token": token.GITHUB_TOKEN,
		"user": map[string]interface{}{
			"id":       session.UserID.String(),
			"username": session.Username,
			"email":    session.Email,
		},
	})
}

func StoreAccessToken(username, githubToken string, user_id uuid.UUID) bool {
	tokenModel := &db.TokenModel{DB: db.DB}

	existingToken, err := tokenModel.GetToken(username)
	if err != nil {
		_, err := tokenModel.SaveToken(githubToken, username, user_id)
		if err != nil {
			fmt.Println("Error: ", err)
			fmt.Println("User created but unable to save github access token")
			return false
		}
		fmt.Println("Saved the github token successfully")
		return true
	}

	fmt.Printf("Token already exists for %s, updating...\n", existingToken.USERNAME)
	err = tokenModel.UpdateToken(username, githubToken)
	if err != nil {
		fmt.Println("Error: ", err)
		return false
	}
	fmt.Println("Updated Successfully")
	return true
}

func GetToken(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["user"]
	super_user_key := vars["super_user_key"]

	if super_user_key != os.Getenv("SECRET_KEY") {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Unauthorized access",
		})
		return
	}

	tokenModel := &db.TokenModel{DB: db.DB}
	github_data, err := tokenModel.GetToken(username)
	if err != nil {
		fmt.Println("Error: ", err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Token not found",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"id":           github_data.ID,
		"username":     github_data.USERNAME,
		"github_token": github_data.GITHUB_TOKEN,
		"user_id":      github_data.USER_ID,
		"created_at":   github_data.CREATED_AT,
	})
}

// Logout - Destroys the user session
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("X-Session-ID")
	if sessionID == "" {
		cookie, err := r.Cookie("session_id")
		if err == nil {
			sessionID = cookie.Value
		}
	}

	if sessionID != "" {
		sessionManager.DeleteSession(sessionID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})
}
