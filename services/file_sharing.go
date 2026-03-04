package services

import (
	"app/urtc/db"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type FileShareRequest struct {
	SenderEmail    string `json:"sender_email"`
	RecipientEmail string `json:"recipient_email"`
	ProjectID      string `json:"project_id"`
	FileName       string `json:"file_name"`
	FileContent    string `json:"file_content"` // Base64 encoded
	FileType       string `json:"file_type"`    // "asset", "script", "scene", etc.
	Message        string `json:"message"`
}

type CodeShareRequest struct {
	SenderEmail    string `json:"sender_email"`
	RecipientEmail string `json:"recipient_email"`
	ProjectID      string `json:"project_id"`
	FileName       string `json:"file_name"`
	Code           string `json:"code"`
	Language       string `json:"language"`
	LineNumber     int    `json:"line_number,omitempty"`
	Message        string `json:"message"`
}

type BulkFileShareRequest struct {
	SenderEmail    string      `json:"sender_email"`
	RecipientEmail string      `json:"recipient_email"`
	ProjectID      string      `json:"project_id"`
	Files          []FileShare `json:"files"`
	Message        string      `json:"message"`
}

type FileShare struct {
	FileName    string `json:"file_name"`
	FileContent string `json:"file_content"`
	FileType    string `json:"file_type"`
}

type ShareableCollaborator struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Status   string `json:"status"`
	IsOnline bool   `json:"is_online"`
}

// ShareFile - Share a single file with a collaborator
func ShareFile(w http.ResponseWriter, r *http.Request) {
	var req FileShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	// Validate required fields
	if req.SenderEmail == "" || req.RecipientEmail == "" || req.FileName == "" || req.FileContent == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Missing required fields",
		})
		return
	}

	// Get sender and recipient users
	userModel := &db.UserModel{DB: db.DB}
	_, err := userModel.GetUserByEmail(req.SenderEmail)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Sender not found",
		})
		return
	}

	recipient, err := userModel.GetUserByEmail(req.RecipientEmail)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Recipient not found",
		})
		return
	}

	// Verify collaboration exists if project_id is provided
	if req.ProjectID != "" {
		projectUUID, err := uuid.Parse(req.ProjectID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Invalid project ID",
			})
			return
		}

		collabModel := &db.CollaboratorModel{DB: db.DB}
		collab, err := collabModel.GetCollaborationByUserAndProject(recipient.ID, projectUUID)
		if err != nil || collab.Status != "approved" {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Recipient is not an approved collaborator",
			})
			return
		}
	}

	// Send file via WebSocket
	metadata := map[string]interface{}{
		"project_id":   req.ProjectID,
		"file_name":    req.FileName,
		"file_content": req.FileContent,
		"file_type":    req.FileType,
		"sender_email": req.SenderEmail,
		"message":      req.Message,
	}

	SendNotificationToUser(
		recipient.ID.String(),
		"file_share",
		req.Message,
		metadata,
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "File shared successfully",
		"file_name": req.FileName,
		"sent_to":   recipient.USERNAME,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// ShareCode - Share code snippet with a collaborator
func ShareCode(w http.ResponseWriter, r *http.Request) {
	var req CodeShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	// Validate required fields
	if req.SenderEmail == "" || req.RecipientEmail == "" || req.Code == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Missing required fields",
		})
		return
	}

	// Get sender and recipient users
	userModel := &db.UserModel{DB: db.DB}
	_, err := userModel.GetUserByEmail(req.SenderEmail)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Sender not found",
		})
		return
	}

	recipient, err := userModel.GetUserByEmail(req.RecipientEmail)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Recipient not found",
		})
		return
	}

	// Send code via WebSocket
	metadata := map[string]interface{}{
		"project_id":   req.ProjectID,
		"file_name":    req.FileName,
		"code":         req.Code,
		"language":     req.Language,
		"line_number":  req.LineNumber,
		"sender_email": req.SenderEmail,
		"message":      req.Message,
	}

	SendNotificationToUser(
		recipient.ID.String(),
		"code_share",
		req.Message,
		metadata,
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "Code shared successfully",
		"sent_to":   recipient.USERNAME,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// ShareBulkFiles - Share multiple files at once
func ShareBulkFiles(w http.ResponseWriter, r *http.Request) {
	var req BulkFileShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	// Validate required fields
	if req.SenderEmail == "" || req.RecipientEmail == "" || len(req.Files) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Missing required fields",
		})
		return
	}

	// Get sender and recipient users
	userModel := &db.UserModel{DB: db.DB}
	_, err := userModel.GetUserByEmail(req.SenderEmail)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Sender not found",
		})
		return
	}

	recipient, err := userModel.GetUserByEmail(req.RecipientEmail)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Recipient not found",
		})
		return
	}

	// Send files via WebSocket
	metadata := map[string]interface{}{
		"project_id":   req.ProjectID,
		"files":        req.Files,
		"file_count":   len(req.Files),
		"sender_email": req.SenderEmail,
		"message":      req.Message,
	}

	SendNotificationToUser(
		recipient.ID.String(),
		"bulk_file_share",
		req.Message,
		metadata,
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"message":    "Files shared successfully",
		"file_count": len(req.Files),
		"sent_to":    recipient.USERNAME,
		"timestamp":  time.Now().Format(time.RFC3339),
	})
}

// GetShareableCollaborators - Get list of approved collaborators for file sharing
func GetShareableCollaborators(w http.ResponseWriter, r *http.Request) {
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

	// Get user model to fetch additional user details
	userModel := &db.UserModel{DB: db.DB}
	shareableCollabs := make([]ShareableCollaborator, 0)

	for _, collab := range collaborators {
		if collab.Status == "approved" {
			user, err := userModel.GetUserByID(collab.UserID)
			if err != nil {
				continue
			}

			shareableCollabs = append(shareableCollabs, ShareableCollaborator{
				UserID:   user.ID.String(),
				Username: user.USERNAME,
				Email:    user.EMAIL,
				Status:   collab.Status,
				IsOnline: manager.isUserOnline(user.ID.String()),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"project_id":    projectID,
		"collaborators": shareableCollabs,
		"total":         len(shareableCollabs),
	})
}
