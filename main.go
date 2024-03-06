package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-chi/chi/v5"
)

type User struct {
	Name    string   `json:"name"`
	Age     int      `json:"age"`
	Friends []string `json:"friends"`
}

var (
	users      = make(map[string]User)
	usersMutex = sync.RWMutex{}
	nextUserID = 1
)

func generateUserID() string {
	id := strconv.Itoa(nextUserID)
	nextUserID++
	return id
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var newUser User
	if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	usersMutex.Lock()
	defer usersMutex.Unlock()

	userID := generateUserID()
	users[userID] = newUser

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "User ID: %s", userID)
}

func getAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	usersMutex.RLock()
	defer usersMutex.RUnlock()

	if len(users) == 0 {
		http.Error(w, "Список пользователей пуст", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(users)
	if err != nil {
		http.Error(w, "Ошибка при формировании ответа", http.StatusInternalServerError)
		return
	}
}

func makeFriendsHandler(w http.ResponseWriter, r *http.Request) {
	var friendship struct {
		SourceID string `json:"source_id"`
		TargetID string `json:"target_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&friendship); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	usersMutex.Lock()
	defer usersMutex.Unlock()

	sourceUser, sourceExists := users[friendship.SourceID]
	targetUser, targetExists := users[friendship.TargetID]

	if !sourceExists || !targetExists {
		http.Error(w, "One or both users not found", http.StatusBadRequest)
		return
	}

	sourceUser.Friends = append(sourceUser.Friends, friendship.TargetID)
	targetUser.Friends = append(targetUser.Friends, friendship.SourceID)

	users[friendship.SourceID] = sourceUser
	users[friendship.TargetID] = targetUser

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s и %s теперь друзья", sourceUser.Name, targetUser.Name)
}

func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		TargetID string `json:"target_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	usersMutex.Lock()
	defer usersMutex.Unlock()

	targetUser, exists := users[request.TargetID]
	if !exists {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}

	delete(users, request.TargetID)

	for _, friendID := range targetUser.Friends {
		friend, ok := users[friendID]
		if !ok {
			continue
		}
		for i, id := range friend.Friends {
			if id == request.TargetID {
				friend.Friends = append(friend.Friends[:i], friend.Friends[i+1:]...)
				break
			}
		}
		users[friendID] = friend
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s удалён", targetUser.Name)
}

func getUserFriendsHandler(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")

	usersMutex.RLock()
	defer usersMutex.RUnlock()

	user, exists := users[userID]
	if !exists {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}

	friendsDetails := []User{}

	for _, friendID := range user.Friends {
		if friend, ok := users[friendID]; ok {
			friendsDetails = append(friendsDetails, friend)
		}
	}

	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(friendsDetails)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func updateUserAgeHandler(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")

	var request struct {
		NewAge int `json:"new_age"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	usersMutex.Lock()
	defer usersMutex.Unlock()

	user, exists := users[userID]
	if !exists {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}

	user.Age = request.NewAge
	users[userID] = user

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Возраст пользователя успешно обновлён")
}

func main() {
	r := chi.NewRouter()

	r.Post("/create", createUserHandler)
	r.Post("/make_friends", makeFriendsHandler)
	r.Delete("/user", deleteUserHandler)
	r.Get("/friends/{user_id}", getUserFriendsHandler)
	r.Get("/users", getAllUsersHandler)
	r.Put("/user_age/{user_id}", updateUserAgeHandler)

	http.ListenAndServe(":8080", r)
}
