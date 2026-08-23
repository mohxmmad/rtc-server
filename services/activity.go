package services

import (
	"app/rtc/db"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type ActivityLog struct {
	ID          uuid.UUID              `json:"id"`
	UserID      uuid.UUID              `json:"user_id"`
	ProjectID   uuid.UUID              `json:"project_id,omitempty"`
	Action      string                 `json:"action"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// LogActivity - Helper function to log user activities
func LogActivity(userID, projectID uuid.UUID, action, description string, metadata map[string]interface{}, r *http.Request) error {
	activityModel := &db.ActivityModel{DB: db.DB}

	ipAddress := r.RemoteAddr
	userAgent := r.UserAgent()

	return activityModel.CreateActivity(userID, projectID, action, description, metadata, ipAddress, userAgent)
}

// GetUserActivities - Retrieves activities for a specific user
func GetUserActivities(w http.ResponseWriter, r *http.Request) {
	userEmail := r.URL.Query().Get("user_email")
	limit := r.URL.Query().Get("limit")

	if userEmail == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "user_email is required",
		})
		return
	}

	userModel := &db.UserModel{DB: db.DB}
	user, err := userModel.GetUserByEmail(userEmail)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "User not found",
		})
		return
	}

	limitInt := 50
	if limit != "" {
		// Parse limit if provided
		if _, err := fmt.Sscanf(limit, "%d", &limitInt); err != nil {
			limitInt = 50
		}
	}

	activityModel := &db.ActivityModel{DB: db.DB}
	activities, err := activityModel.GetUserActivities(user.ID, limitInt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to fetch activities",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"user_email": userEmail,
		"activities": activities,
		"total":      len(activities),
	})
}

// GetProjectActivities - Retrieves activities for a specific project
func GetProjectActivities(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	limit := r.URL.Query().Get("limit")

	if projectID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "project_id is required",
		})
		return
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid project ID",
		})
		return
	}

	limitInt := 50
	if limit != "" {
		if _, err := fmt.Sscanf(limit, "%d", &limitInt); err != nil {
			limitInt = 50
		}
	}

	activityModel := &db.ActivityModel{DB: db.DB}
	activities, err := activityModel.GetProjectActivities(projectUUID, limitInt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to fetch activities",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"project_id": projectID,
		"activities": activities,
		"total":      len(activities),
	})
}

// GetRecentTeamActivities - Get recent activities across all team members
func GetRecentTeamActivities(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	limit := r.URL.Query().Get("limit")

	if projectID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "project_id is required",
		})
		return
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid project ID",
		})
		return
	}

	limitInt := 100
	if limit != "" {
		if _, err := fmt.Sscanf(limit, "%d", &limitInt); err != nil {
			limitInt = 100
		}
	}

	activityModel := &db.ActivityModel{DB: db.DB}
	activities, err := activityModel.GetProjectActivities(projectUUID, limitInt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to fetch team activities",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"project_id": projectID,
		"activities": activities,
		"total":      len(activities),
	})
}
