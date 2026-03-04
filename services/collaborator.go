package services

import (
	"app/urtc/db"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type CollabRequest struct {
	OwnerEmail        string `json:"owner_email"`
	CollaboratorEmail string `json:"collaborator_email"`
	ProjectID         string `json:"project_id"`
}

type CollabApproval struct {
	CollabID string `json:"collab_id"`
	Status   string `json:"status"` // "approved" or "rejected"
}

type CollabNotification struct {
	CollabID          string `json:"collab_id"`
	ProjectID         string `json:"project_id"`
	ProjectName       string `json:"project_name"`
	ProjectOwner      string `json:"project_owner"`
	CollaboratorEmail string `json:"collaborator_email"`
	CollaboratorName  string `json:"collaborator_name"`
	Status            string `json:"status"`
	CreatedAt         string `json:"created_at"`
}

type PendingCollabDetail struct {
	CollabID          string `json:"collab_id"`
	CollaboratorID    string `json:"collaborator_id"`
	CollaboratorName  string `json:"collaborator_name"`
	CollaboratorEmail string `json:"collaborator_email"`
	ProjectID         string `json:"project_id"`
	ProjectName       string `json:"project_name"`
	Status            string `json:"status"`
	RequestedAt       string `json:"requested_at"`
}

// RequestCollaboration - Creates a collaboration request with pending status
func RequestCollaboration(w http.ResponseWriter, r *http.Request) {
	var req CollabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	// Validate project exists and owner has access
	userModel := &db.UserModel{DB: db.DB}
	owner, err := userModel.GetUserByEmail(req.OwnerEmail)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Owner not found",
		})
		return
	}

	// Verify project belongs to owner
	projectModel := &db.ProjectModel{DB: db.DB}
	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid project ID",
		})
		return
	}

	project, err := projectModel.GetProjectByID(projectID)
	if err != nil || project.OwnerID != owner.ID {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Project not found or access denied",
		})
		return
	}

	// Check if collaborator exists and is GitHub authenticated
	collaborator, err := userModel.GetUserByEmail(req.CollaboratorEmail)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Collaborator not found. User must sign in with GitHub first",
		})
		return
	}

	// Verify collaborator has GitHub token (authenticated)
	tokenModel := &db.TokenModel{DB: db.DB}
	_, err = tokenModel.GetToken(collaborator.USERNAME)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Collaborator must authenticate with GitHub first",
		})
		return
	}

	// Create collaboration request with PENDING status
	collabModel := &db.CollaboratorModel{DB: db.DB}
	collab, err := collabModel.CreateCollaboration(collaborator.ID, projectID, "pending")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to create collaboration request: " + err.Error(),
		})
		return
	}

	// Send notification to collaborator
	notification := CollabNotification{
		CollabID:          collab.ID.String(),
		ProjectID:         project.ID.String(),
		ProjectName:       project.Name,
		ProjectOwner:      owner.USERNAME,
		CollaboratorEmail: collaborator.EMAIL,
		CollaboratorName:  collaborator.USERNAME,
		Status:            collab.Status,
		CreatedAt:         collab.CreatedAt.String(),
	}

	fmt.Printf("\n=== COLLABORATION REQUEST NOTIFICATION ===\n")
	fmt.Printf("To: %s\n", notification.CollaboratorEmail)
	fmt.Printf("From: %s\n", notification.ProjectOwner)
	fmt.Printf("Project: %s (ID: %s)\n", notification.ProjectName, notification.ProjectID)
	fmt.Printf("Collaboration ID: %s\n", notification.CollabID)
	fmt.Printf("Status: %s\n", notification.Status)
	fmt.Printf("==========================================\n\n")

	// Add email logic here to notify collaborator in real implementation

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "Collaboration request sent successfully",
		"collab_id": collab.ID,
		"status":    collab.Status,
		//"notification": notification,
	})
}

// ApproveCollaboration - Approves/Rejects collaboration using SESSION-based authentication
func ApproveCollaboration(w http.ResponseWriter, r *http.Request) {
	// Get session from context (set by SessionAuth middleware)
	session, ok := GetSessionFromContext(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Session not found. Please login first.",
		})
		return
	}

	var req CollabApproval
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	// Validate status
	if req.Status != "approved" && req.Status != "rejected" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Status must be 'approved' or 'rejected'",
		})
		return
	}

	// Validate collab ID
	collabID, err := uuid.Parse(req.CollabID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid collaboration ID",
		})
		return
	}

	// Get collaboration details
	collabModel := &db.CollaboratorModel{DB: db.DB}
	collab, err := collabModel.GetCollaborationByID(collabID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Collaboration request not found",
		})
		return
	}

	// Get collaborator user
	userModel := &db.UserModel{DB: db.DB}
	collaborator, err := userModel.GetUserByID(collab.UserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to fetch collaborator details",
		})
		return
	}

	// Authenticate collaborator using token
	tokenModel := &db.TokenModel{DB: db.DB}
	storedToken, err := tokenModel.GetToken(collaborator.USERNAME)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Collaborator authentication failed",
		})
		return
	} else {
		fmt.Println("Stored Token: ", storedToken.GITHUB_TOKEN)
		// In real implementation, verify token validity with session ID implementation via middleware to be added
	}

	// Update collaboration status
	err = collabModel.UpdateCollaborationStatus(collabID, req.Status)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to update collaboration status",
		})
		return
	}

	// Get project details for notification
	projectModel := &db.ProjectModel{DB: db.DB}
	project, _ := projectModel.GetProjectByID(collab.ProjectID)

	fmt.Printf("\n=== COLLABORATION %s ===\n", req.Status)
	fmt.Printf("Collaboration ID: %s\n", req.CollabID)
	fmt.Printf("Project: %s\n", project.Name)
	fmt.Printf("Collaborator: %s (%s)\n", session.Username, session.Email)
	fmt.Printf("New Status: %s\n", req.Status)
	fmt.Printf("==========================================\n\n")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   fmt.Sprintf("Collaboration request %s", req.Status),
		"collab_id": req.CollabID,
		"status":    req.Status,
	})
}

// GetProjectPendingCollaborations - Owner gets all pending collab requests for their project
// Requires session authentication
func GetProjectPendingCollaborations(w http.ResponseWriter, r *http.Request) {
	// Get session from context
	session, ok := GetSessionFromContext(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Session not found. Please login first.",
		})
		return
	}

	// Get project name from query parameter
	projectName := r.URL.Query().Get("project_name")
	if projectName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "project_name is required",
		})
		return
	}

	// Get project by name and verify ownership
	projectModel := &db.ProjectModel{DB: db.DB}
	project, err := projectModel.GetProjectByName(session.UserID, projectName)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Project not found or you are not the owner",
		})
		return
	}

	// Get all collaborators for the project
	collabModel := &db.CollaboratorModel{DB: db.DB}
	collaborators, err := collabModel.GetProjectCollaborators(project.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to fetch collaboration requests",
		})
		return
	}

	// Filter pending requests and get user details
	userModel := &db.UserModel{DB: db.DB}
	pendingRequests := []PendingCollabDetail{}

	for _, collab := range collaborators {
		if collab.Status == "pending" {
			user, err := userModel.GetUserByID(collab.UserID)
			if err != nil {
				continue
			}

			pendingRequests = append(pendingRequests, PendingCollabDetail{
				CollabID:          collab.ID.String(),
				CollaboratorID:    user.ID.String(),
				CollaboratorName:  user.USERNAME,
				CollaboratorEmail: user.EMAIL,
				ProjectID:         project.ID.String(),
				ProjectName:       project.Name,
				Status:            collab.Status,
				RequestedAt:       collab.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"project_name": projectName,
		"project_id":   project.ID.String(),
		"owner":        session.Username,
		"requests":     pendingRequests,
		"total":        len(pendingRequests),
	})
}

// GetProjectCollaborators - Lists all collaborators for a project
func GetProjectCollaborators(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
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

	collabModel := &db.CollaboratorModel{DB: db.DB}
	collaborators, err := collabModel.GetProjectCollaborators(projectUUID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to fetch collaborators",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"project_id":    projectID,
		"collaborators": collaborators,
		"total":         len(collaborators),
	})
}

// GetUserCollaborationRequests - Lists pending collaboration requests for a user
func GetUserCollaborationRequests(w http.ResponseWriter, r *http.Request) {
	userEmail := r.URL.Query().Get("user_email")
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

	collabModel := &db.CollaboratorModel{DB: db.DB}
	requests, err := collabModel.GetUserPendingRequests(user.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to fetch collaboration requests",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"user_email": userEmail,
		"requests":   requests,
		"total":      len(requests),
	})
}

// RemoveCollaborator - Removes a collaborator from a project
func RemoveCollaborator(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collabID := vars["collab_id"]

	collabUUID, err := uuid.Parse(collabID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid collaboration ID",
		})
		return
	}

	collabModel := &db.CollaboratorModel{DB: db.DB}
	err = collabModel.DeleteCollaboration(collabUUID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to remove collaborator",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Collaborator removed successfully",
	})
}
