package services

import (
	"app/urtc/db"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

type MetaUser struct {
	EMAIL        string `json:"user_email"`
	PROJECT_NAME string `json:"project_name"`
}

// type CollabRequest struct {
// 	CollaboratorEmail string `json:"collaborator_email"`
// 	ProjectID         string `json:"project_id"`
// }

// type CollabApproval struct {
// 	CollabID string `json:"collab_id"`
// 	Status   string `json:"status"`
// }

func PushProject(w http.ResponseWriter, r *http.Request) {

	var metaUser MetaUser
	json.NewDecoder(r.Body).Decode(&metaUser)

	userModel := &db.UserModel{
		DB: db.DB,
	}
	user, err := userModel.GetUserByEmail(metaUser.EMAIL)
	if err != nil {
		fmt.Println("Error : ", err)
		w.WriteHeader(http.StatusBadRequest)
	} else {
		fmt.Println(user.USERNAME, user.ID, user.GITHUB_ID, user.EMAIL, user.CREATED_AT)

		//starting a new project from here
		projectModel := &db.ProjectModel{
			DB: db.DB,
		}

		tokenModel := &db.TokenModel{
			DB: db.DB,
		}

		token, err := tokenModel.GetToken(user.USERNAME)
		if err != nil {
			fmt.Println("Error Fetching Token : ", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		//Check if a project by this name already exists
		project, err := projectModel.GetProjectByName(user.ID, metaUser.PROJECT_NAME)
		if err != nil {
			project, err := projectModel.CreateProject(user.ID, metaUser.PROJECT_NAME, metaUser.PROJECT_NAME)
			if err != nil {
				fmt.Println("Error : ", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
			fmt.Println("Project Created Succesfully")

			//Start Initializing a github repo here
			payload := map[string]interface{}{
				"name":    metaUser.PROJECT_NAME,
				"private": false,
			}

			body, _ := json.Marshal(payload)

			req, _ := http.NewRequest("POST", "https://api.github.com/user/repos", bytes.NewBuffer(body))
			req.Header.Set("Authorization", "token "+token.GITHUB_TOKEN)
			req.Header.Set("Accept", "application/vnd.github.v3+json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == 201 {
				fmt.Println("Repository created successfully!")
			} else {
				fmt.Println("Failed to create repository. Status:", resp.Status)
			}

			var repos struct {
				Name    string `json:"name"`
				HTMLURL string `json:"html_url"` // This is the repo link
			}

			if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
				http.Error(w, "Failed to parse repos", http.StatusInternalServerError)
				return
			}

			fmt.Fprintln(w, "success : ", http.StatusOK)
			fmt.Fprintln(w, "message : Collaboration started successfully for project ", project.Name)
			fmt.Fprintln(w, "project_id : ", project.ID)
			fmt.Fprintf(w, "github repo URL: %s", repos.HTMLURL)

		} else {
			fmt.Println("Project already exists with the name : ", project.Name)
			w.WriteHeader(http.StatusBadRequest)
		}
	}

}

// func RequestCollaboration(w http.ResponseWriter, r *http.Request) {
// 	var req CollabRequest
// 	json.NewDecoder(r.Body).Decode(&req)

// 	userModel := &db.UserModel{DB: db.DB}
// 	user, err := userModel.GetUserByEmail(req.CollaboratorEmail)
// 	if err != nil {
// 		w.WriteHeader(http.StatusNotFound)
// 		fmt.Fprintln(w, "User not found")
// 		return
// 	}

// 	query := `INSERT INTO collaborators (user_id, project_id) VALUES ($1, $2) RETURNING id`
// 	var collabID string
// 	err = db.DB.QueryRow(query, user.ID, req.ProjectID).Scan(&collabID)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		fmt.Fprintln(w, "Error creating request")
// 		return
// 	}

// 	fmt.Fprintf(w, "Collaboration request created: %s", collabID)
// }

// func ApproveCollaboration(w http.ResponseWriter, r *http.Request) {
// 	var req CollabApproval
// 	json.NewDecoder(r.Body).Decode(&req)

// 	query := `UPDATE collaborators SET status = $1 WHERE id = $2`
// 	_, err := db.DB.Exec(query, req.Status, req.CollabID)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		fmt.Fprintln(w, "Error updating status")
// 		return
// 	}

// 	fmt.Fprintf(w, "Collaboration %s", req.Status)
// }

// func GetProjectCollaborators(w http.ResponseWriter, r *http.Request) {
// 	projectID := r.URL.Query().Get("project_id")
// 	if projectID == "" {
// 		w.WriteHeader(http.StatusBadRequest)
// 		fmt.Fprintln(w, "project_id is required")
// 		return
// 	}

// 	query := `
// 		SELECT c.id, c.status, c.created_at, u.username, u.email
// 		FROM collaborators c
// 		JOIN users u ON c.user_id = u.id
// 		WHERE c.project_id = $1
// 		ORDER BY c.created_at DESC
// 	`

// 	rows, err := db.DB.Query(query, projectID)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		fmt.Fprintln(w, "Error fetching collaborators")
// 		return
// 	}
// 	defer rows.Close()

// 	var collaborators []map[string]interface{}
// 	for rows.Next() {
// 		var id, status, username, email string
// 		var createdAt string
// 		err := rows.Scan(&id, &status, &createdAt, &username, &email)
// 		if err != nil {
// 			continue
// 		}
// 		collaborators = append(collaborators, map[string]interface{}{
// 			"collab_id":  id,
// 			"status":     status,
// 			"username":   username,
// 			"email":      email,
// 			"created_at": createdAt,
// 		})
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(map[string]interface{}{
// 		"project_id":    projectID,
// 		"collaborators": collaborators,
// 		"total":         len(collaborators),
// 	})
// }

// func GetUserCollaborationRequests(w http.ResponseWriter, r *http.Request) {
// 	userEmail := r.URL.Query().Get("user_email")
// 	if userEmail == "" {
// 		w.WriteHeader(http.StatusBadRequest)
// 		fmt.Fprintln(w, "user_email is required")
// 		return
// 	}

// 	userModel := &db.UserModel{DB: db.DB}
// 	user, err := userModel.GetUserByEmail(userEmail)
// 	if err != nil {
// 		w.WriteHeader(http.StatusNotFound)
// 		fmt.Fprintln(w, "User not found")
// 		return
// 	}

// 	query := `
// 		SELECT c.id, c.status, c.created_at, p.name, p.description, u.username as owner_username
// 		FROM collaborators c
// 		JOIN projects p ON c.project_id = p.id
// 		JOIN users u ON p.owner_id = u.id
// 		WHERE c.user_id = $1 AND c.status = 'pending'
// 		ORDER BY c.created_at DESC
// 	`

// 	rows, err := db.DB.Query(query, user.ID)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		fmt.Fprintln(w, "Error fetching requests")
// 		return
// 	}
// 	defer rows.Close()

// 	var requests []map[string]interface{}
// 	for rows.Next() {
// 		var collabID, status, createdAt, projectName, projectDesc, ownerUsername string
// 		err := rows.Scan(&collabID, &status, &createdAt, &projectName, &projectDesc, &ownerUsername)
// 		if err != nil {
// 			continue
// 		}
// 		requests = append(requests, map[string]interface{}{
// 			"collab_id":      collabID,
// 			"status":         status,
// 			"project_name":   projectName,
// 			"description":    projectDesc,
// 			"owner_username": ownerUsername,
// 			"created_at":     createdAt,
// 		})
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(map[string]interface{}{
// 		"user_email": userEmail,
// 		"requests":   requests,
// 		"total":      len(requests),
// 	})
// }

func JoinCollaboration(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Attempting to join collaboration...")
	var req struct {
		UserEmail string `json:"user_email"`
		Token     string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	collabID, err := uuid.Parse(req.Token)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid token format"})
		return
	}

	collabModel := &db.CollaboratorModel{DB: db.DB}
	err = collabModel.UpdateCollaborationStatus(collabID, "approved")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to approve collaboration: " + err.Error()})
		return
	}

	collab, err := collabModel.GetCollaborationByID(collabID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch collaboration details"})
		return
	}

	projectModel := &db.ProjectModel{DB: db.DB}
	project, err := projectModel.GetProjectByID(collab.ProjectID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch project details"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"message":    "Joined collaboration successfully",
		"project_id": project.ID.String(),
		"repo_url":   "https://github.com/TODO/" + project.Name,
	})
}
