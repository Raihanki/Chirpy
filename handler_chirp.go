package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/Raihanki/Chirpy/internal/database"
)

type Chirp struct {
	ID       int    `json:"id"`
	Body     string `json:"body"`
	AuthorId int    `json:"author_id"`
}

func (cfg *apiConfig) handlerChirpsCreate(w http.ResponseWriter, r *http.Request) {
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

	userId, err := claims.GetSubject()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get subject from claims")
		return
	}

	userIdInt, _ := strconv.Atoi(userId)

	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	cleaned, err := validateChirp(params.Body)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	chirp, err := cfg.DB.CreateChirp(cleaned, userIdInt)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create chirp")
		return
	}

	respondWithJSON(w, http.StatusCreated, Chirp{
		ID:       chirp.Id,
		Body:     chirp.Body,
		AuthorId: userIdInt,
	})
}

func validateChirp(body string) (string, error) {
	const maxChirpLength = 140
	if len(body) > maxChirpLength {
		return "", errors.New("Chirp is too long")
	}

	badWords := map[string]struct{}{
		"kerfuffle": {},
		"sharbert":  {},
		"fornax":    {},
	}
	cleaned := getCleanedBody(body, badWords)
	return cleaned, nil
}

func getCleanedBody(body string, badWords map[string]struct{}) string {
	words := strings.Split(body, " ")
	for i, word := range words {
		loweredWord := strings.ToLower(word)
		if _, ok := badWords[loweredWord]; ok {
			words[i] = "****"
		}
	}
	cleaned := strings.Join(words, " ")
	return cleaned
}

func (cfg *apiConfig) handlerDetailChirp(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("chirpId")
	chirpId, err := strconv.Atoi(id)
	if err != nil {
		log.Printf("Error converting int to string: %v", err)
		w.WriteHeader(500)
		return
	}

	data, err := cfg.DB.LoadDB()
	if err != nil {
		log.Printf("Error load DB: %v", err)
		w.WriteHeader(500)
		return
	}

	var getChirp database.Chirp
	for _, chirp := range data.Chirps {
		if chirp.Id == chirpId {
			getChirp = chirp
		}
	}

	if getChirp == (database.Chirp{}) {
		w.WriteHeader(404)
		return
	}

	chirpJson, err := json.Marshal(getChirp)
	if err != nil {
		log.Printf("Error marshal chirp: %v", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(200)
	w.Write(chirpJson)
}

func (cfg *apiConfig) handlerChirpsRetrieve(w http.ResponseWriter, r *http.Request) {
	author_id := r.URL.Query().Get("author_id")
	sortFilter := r.URL.Query().Get("sort")

	if sortFilter == "" {
		sortFilter = "asc"
	}

	dbChirps, err := cfg.DB.GetChirps(author_id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't retrieve chirps")
		return
	}

	chirps := []Chirp{}
	for _, dbChirp := range dbChirps {
		chirps = append(chirps, Chirp{
			ID:       dbChirp.Id,
			Body:     dbChirp.Body,
			AuthorId: dbChirp.AuthorId,
		})
	}

	sort.Slice(chirps, func(i, j int) bool {
		if sortFilter == "desc" {
			// Sort in descending order
			return chirps[i].ID > chirps[j].ID
		}
		// Default to ascending order
		return chirps[i].ID < chirps[j].ID
	})

	respondWithJSON(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	chirpIdVal := r.PathValue("chirpId")
	chirpId, err := strconv.Atoi(chirpIdVal)
	if err != nil {
		log.Printf("Error converting int to string: %v", err)
		w.WriteHeader(500)
		return
	}

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

	userId, err := claims.GetSubject()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get subject from claims")
		return
	}

	userIdInt, _ := strconv.Atoi(userId)

	chirp, err := cfg.DB.GetChirpById(chirpId)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get chirp")
		return
	}

	if userIdInt != chirp.AuthorId {
		w.WriteHeader(403)
		return
	}

	err = cfg.DB.DeleteChirp(chirp)
	if err != nil {
		log.Fatalf("erro delete chirp: %v", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(204)
}
