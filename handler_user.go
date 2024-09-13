package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Raihanki/Chirpy/internal/database"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID          int    `json:"id"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	IsChirpyRed bool   `json:"is_chirpy_red"`
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type UserRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	request := UserRequest{}
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Printf("Error while decode request body: %v", err)
		w.WriteHeader(500)
		return
	}

	newUser, err := cfg.DB.CreateUser(request.Email, request.Password)
	if err != nil {
		log.Printf("Error while creating user: %v", err)
		w.WriteHeader(500)
		return
	}

	jsonUser, err := json.Marshal(newUser)
	if err != nil {
		log.Printf("Error while mrshal user: %v", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(201)
	w.Write(jsonUser)
}

func (cfg *apiConfig) handlerUserLogin(w http.ResponseWriter, r *http.Request) {
	type LoginRequest struct {
		Email            string `json:"email"`
		Password         string `json:"password"`
		ExpiresInSeconds *int   `json:"expires_in_seconds,omitempty"`
	}

	request := LoginRequest{}
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Printf("Error while decoding request: %v", err)
		w.WriteHeader(500)
		return
	}

	data, err := cfg.DB.LoadDB()
	if err != nil {
		log.Printf("Error while load database: %v", err)
		w.WriteHeader(500)
		return
	}

	var user database.User
	for _, u := range data.Users {
		if u.Email == request.Email {
			user = u
		}
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password))
	if err != nil {
		w.WriteHeader(401)
		return
	}

	type UserResponse struct {
		ID           int    `json:"id"`
		Email        string `json:"email"`
		Password     string `json:"-"`
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
		IsChirpyRed  bool   `json:"is_chirpy_red"`
	}

	exp := 0
	if request.ExpiresInSeconds == nil {
		exp = 86400
	} else {
		exp = *request.ExpiresInSeconds
	}

	strUserId := strconv.Itoa(user.ID)
	jwtConfig := JwtConfig{
		Issuer:    "chirpy",
		ExpiresAt: exp,
		Subject:   strUserId,
	}

	token, err := jwtConfig.generateToken()
	if err != nil {
		log.Printf("error generat token: %v", err)
		w.WriteHeader(500)
		return
	}

	// Create a byte slice to hold the random data (32 bytes for 256 bits)
	refreshToken := make([]byte, 32)

	// Read random bytes using crypto/rand's rand.Read function
	_, err = rand.Read(refreshToken)
	if err != nil {
		log.Printf("error generat refresh token: %v", err)
		w.WriteHeader(500)
		return
	}
	rToken := hex.EncodeToString(refreshToken)
	cfg.DB.SaveRefreshToken(user.ID, rToken)

	jsonUser, err := json.Marshal(UserResponse{
		ID:           user.ID,
		Email:        user.Email,
		Password:     user.Password,
		Token:        token,
		RefreshToken: rToken,
		IsChirpyRed:  user.IsChirpyRed,
	})

	if err != nil {
		log.Printf("error marshaling user data")
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(200)
	w.Write(jsonUser)
}

func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")
	if header == "" || !strings.HasPrefix(header, "Bearer ") {
		log.Println("Authorization header is missing or improperly formatted")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	token := strings.Split(header, " ")[1]

	claims, err := ValidateToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	type UserRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	request := UserRequest{}
	err = json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Fatalf("error decode body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	userId, err := claims.GetSubject()
	if err != nil {
		log.Fatalf("error get subject: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userIdInt, _ := strconv.Atoi(userId)

	updatedUser, err := cfg.DB.UpdateUser(request.Email, request.Password, userIdInt)
	if err != nil {
		log.Fatalf("error updating user %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonUser, err := json.Marshal(updatedUser)
	if err != nil {
		log.Fatalf("error updating user %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(200)
	w.Write(jsonUser)
}

func (cfg *apiConfig) handlerRefreshToken(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")
	if header == "" || !strings.HasPrefix(header, "Bearer ") {
		log.Println("Authorization header is missing or improperly formatted")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	token := strings.Split(header, " ")[1]

	user, err := cfg.DB.ValidateRefreshToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	strUserId := strconv.Itoa(user.ID)
	jwtConfig := JwtConfig{
		Issuer:    "chirpy",
		ExpiresAt: 100,
		Subject:   strUserId,
	}
	newToken, err := jwtConfig.generateToken()
	if err != nil {
		log.Fatalf("Error create token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type Response struct {
		Token string `json:"token"`
	}
	response := Response{
		Token: newToken,
	}
	tokenResponse, err := json.Marshal(response)
	if err != nil {
		log.Fatalf("Error marshal data: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(200)
	w.Write(tokenResponse)
}

func (cfg *apiConfig) handlerRevokeToken(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")
	if header == "" || !strings.HasPrefix(header, "Bearer ") {
		log.Println("Authorization header is missing or improperly formatted")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	token := strings.Split(header, " ")[1]

	user, err := cfg.DB.ValidateRefreshToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err = cfg.DB.DeleteRefreshToken(user)
	if err != nil {
		log.Fatalf("error delete refresh token : %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(204)
}

func (cfg *apiConfig) handlerPolkaWebhook(w http.ResponseWriter, r *http.Request) {
	type UserInfo struct {
		UserId int `json:"user_id"`
	}

	type RequestBody struct {
		Event string   `json:"event"`
		Data  UserInfo `json:"data"`
	}

	header := r.Header.Get("Authorization")
	if header == "" || !strings.HasPrefix(header, "ApiKey ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	token := strings.Split(header, " ")[1]

	if token != os.Getenv("POLKA_API_KEY") {
		w.WriteHeader(401)
	}

	request := RequestBody{}
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Fatalf("error encoding request body : %v", err)
		w.WriteHeader(500)
		return
	}

	if request.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}

	err = cfg.DB.UpgradeChirpy(request.Data.UserId)
	if err != nil {
		if errors.Is(err, errors.New("notfound")) {
			w.WriteHeader(404)
		}
		log.Printf("LOGGG:: eror: %v", err)
	}

	w.WriteHeader(204)
}
