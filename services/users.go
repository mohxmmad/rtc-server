package services

import (
	"app/rtc/db"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func GetUsersLen(w http.ResponseWriter, r *http.Request) {
	userModel := &db.UserModel{
		DB: db.DB,
	}
	users, err := userModel.GetAllUsers()
	if err != nil {
		fmt.Println("Error : ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "No of Users : %d", len(users))

}

func GetUsers(w http.ResponseWriter, r *http.Request) {
	userModel := &db.UserModel{
		DB: db.DB,
	}
	users, err := userModel.GetAllUsers()
	if err != nil {
		fmt.Println("Error : ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	for i := 0; i < len(users); i++ {
		user := users[i]
		fmt.Fprintf(w, "ID : %d , Username : %s, Github Id : %d, Email : %s, Created At : %s \n", user.ID, user.USERNAME, user.GITHUB_ID, user.EMAIL, user.CREATED_AT)
	}

}

func GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["user"]
	userModel := &db.UserModel{
		DB: db.DB,
	}
	user, err := userModel.GetUser(username)
	if err != nil {
		fmt.Println("Error : ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "ID : %d , Username : %s, Github Id : %d, Email : %s, Created At : %s \n", user.ID, user.USERNAME, user.GITHUB_ID, user.EMAIL, user.CREATED_AT)

}

func GetUserByEmail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	email := vars["email"]
	userModel := &db.UserModel{
		DB: db.DB,
	}
	user, err := userModel.GetUserByEmail(email)
	if err != nil {
		fmt.Println("Error : ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "ID : %d , Username : %s, Github Id : %d, Email : %s, Created At : %s \n", user.ID, user.USERNAME, user.GITHUB_ID, user.EMAIL, user.CREATED_AT)

}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["user"]
	userModel := &db.UserModel{
		DB: db.DB,
	}
	err := userModel.DeleteUser(username)
	if err != nil {
		fmt.Println("Error : ", err)
		w.WriteHeader(http.StatusExpectationFailed)

	}

}
