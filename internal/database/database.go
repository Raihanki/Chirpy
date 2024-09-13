package database

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
)

type DB struct {
	path string
	mu   *sync.RWMutex
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
	Users  map[int]User  `json:"users"`
}

func NewDB(path string) (*DB, error) {
	db := &DB{
		path: path,
		mu:   &sync.RWMutex{},
	}

	err := db.ensureDB()
	return db, err
}

func (db *DB) WriteDB(dbStructure DBStructure) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	data, errMarshal := json.Marshal(dbStructure)
	if errMarshal != nil {
		return errMarshal
	}

	errWriteFile := os.WriteFile(db.path, data, 0600)
	if errWriteFile != nil {
		return errWriteFile
	}

	return nil
}

func (db *DB) ensureDB() error {
	_, errReadFile := os.ReadFile(db.path)
	if errors.Is(errReadFile, os.ErrNotExist) {
		dbStructure := DBStructure{
			Chirps: map[int]Chirp{},
			Users:  map[int]User{},
		}
		return db.WriteDB(dbStructure)
	}

	return errReadFile
}

func (db *DB) LoadDB() (DBStructure, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	dbStructure := DBStructure{}
	data, errReadFile := os.ReadFile(db.path)
	if errReadFile != nil {
		if errors.Is(os.ErrNotExist, errReadFile) {
			return dbStructure, nil
		}
	}

	errUnmarshal := json.Unmarshal(data, &dbStructure)
	if errUnmarshal != nil {
		return DBStructure{}, nil
	}

	return dbStructure, nil
}
