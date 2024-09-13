package database

import (
	"errors"
	"strconv"
)

type Chirp struct {
	Id       int    `json:"id"`
	Body     string `json:"body"`
	AuthorId int    `json:"author_id"`
}

func (db *DB) CreateChirp(body string, userId int) (Chirp, error) {
	data, errLoadDb := db.LoadDB()
	if errLoadDb != nil {
		return Chirp{}, errLoadDb
	}

	newId := len(data.Chirps) + 1
	chirp := Chirp{
		Id:       newId,
		Body:     body,
		AuthorId: userId,
	}
	data.Chirps[newId] = chirp

	errWriteDb := db.WriteDB(data)
	if errWriteDb != nil {
		return Chirp{}, errWriteDb
	}

	return chirp, nil
}

func (db *DB) GetChirps(author_id string) ([]Chirp, error) {
	data, err := db.LoadDB()
	if err != nil {
		return []Chirp{}, err
	}

	var chirps []Chirp
	if author_id == "" {
		for _, chirp := range data.Chirps {
			chirps = append(chirps, chirp)
		}
	} else {
		for _, chirp := range data.Chirps {
			intAuthorId, _ := strconv.Atoi(author_id)
			if chirp.AuthorId == intAuthorId {
				chirps = append(chirps, chirp)
			}
		}
	}
	return chirps, nil
}

func (db *DB) GetChirpById(id int) (Chirp, error) {
	data, err := db.LoadDB()
	if err != nil {
		return Chirp{}, err
	}

	var chirp Chirp
	for _, c := range data.Chirps {
		if c.Id == id {
			chirp = c
			break
		}
	}

	return chirp, nil
}

func (db *DB) DeleteChirp(chirp Chirp) error {
	data, err := db.LoadDB()
	if err != nil {
		return err
	}

	if _, exists := data.Chirps[chirp.Id]; !exists {
		return errors.New("chirp not found")
	}

	delete(data.Chirps, chirp.Id)

	return db.WriteDB(data)
}
