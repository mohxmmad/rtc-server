package services

import (
	"app/urtc/db"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func NProjects(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["owner"]
	userModel := &db.UserModel{
		DB: db.DB,
	}
	user, err := userModel.GetUser(username)
	if err != nil {
		fmt.Println("Invalid Username")
		w.WriteHeader(http.StatusBadRequest)
	} else {
		owner_id := user.ID
		projectModel := &db.ProjectModel{
			DB: db.DB,
		}
		projects, err := projectModel.GetProjectsByUser(owner_id)
		if err != nil {
			fmt.Println("Error : ", err)
			w.WriteHeader(http.StatusBadRequest)
		}
		fmt.Fprintf(w, "No of Projects : %d", len(projects))
	}
}

func GetProjects(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["owner"]
	userModel := &db.UserModel{
		DB: db.DB,
	}
	user, err := userModel.GetUser(username)
	if err != nil {
		fmt.Println("Invalid Username")
		w.WriteHeader(http.StatusBadRequest)
	} else {
		owner_id := user.ID
		projectModel := &db.ProjectModel{
			DB: db.DB,
		}
		projects, err := projectModel.GetProjectsByUser(owner_id)
		if err != nil {
			fmt.Println("Error : ", err)
			w.WriteHeader(http.StatusBadRequest)
		}

		for i := 0; i < len(projects); i++ {
			project := projects[i]
			fmt.Fprintf(w, "ID : %s , Owner ID : %s, Name : %s, Description : %s, Created At : %s \n", project.ID.String(), project.OwnerID.String(), project.Name, project.Description, project.CreatedAt)
		}
	}
}
func GetProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["owner"]
	nameStr := vars["name"]

	userModel := &db.UserModel{
		DB: db.DB,
	}
	user, err := userModel.GetUser(username)
	if err != nil {
		http.Error(w, "Invalid owner ID", http.StatusBadRequest)
		w.WriteHeader(http.StatusBadRequest)
		return
	} else {
		ownerID := user.ID
		projectModel := &db.ProjectModel{
			DB: db.DB,
		}
		project, err := projectModel.GetProjectByName(ownerID, nameStr)
		if err != nil {
			fmt.Println("Error : ", err)
			w.WriteHeader(http.StatusBadRequest)

		}

		fmt.Fprintf(w, "ID : %s , Owner ID : %s, Name : %s, Description : %s, Created At : %s \n", project.ID.String(), project.OwnerID.String(), project.Name, project.Description, project.CreatedAt)
	}
}

func DeleteProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["owner"]
	projectName := vars["name"]

	userModel := &db.UserModel{
		DB: db.DB,
	}
	user, err := userModel.GetUser(username)
	if err != nil {
		http.Error(w, "Invalid owner ID", http.StatusBadRequest)
		w.WriteHeader(http.StatusBadRequest)

		return
	} else {
		ownerID := user.ID
		projectModel := &db.ProjectModel{
			DB: db.DB,
		}
		project, err := projectModel.GetProjectByName(ownerID, projectName)
		if err != nil {
			fmt.Println("Error : ", err)
			w.WriteHeader(http.StatusBadRequest)

		}
		projectID := project.ID
		err = projectModel.DeleteProject(ownerID, projectID)
		if err != nil {
			fmt.Println("Error : ", err)
		}
	}
}
