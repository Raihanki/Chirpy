package database

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int    `json:"id"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	RefreshToken string `json:"refresh_token"`
	IsChirpyRed  bool   `json:"is_chirpy_red"`
}

func (db *DB) CreateUser(email string, password string) (User, error) {
	data, err := db.LoadDB()
	if err != nil {
		return User{}, nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}

	newId := len(data.Users) + 1
	user := User{
		ID:          newId,
		Email:       email,
		Password:    string(hashedPassword),
		IsChirpyRed: false,
	}
	data.Users[newId] = user

	dbStructure := DBStructure{
		Chirps: data.Chirps,
		Users:  data.Users,
	}
	err = db.WriteDB(dbStructure)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (db *DB) UpdateUser(email string, password string, userId int) (User, error) {
	data, err := db.LoadDB()
	if err != nil {
		return User{}, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}

	var updatedUser User
	if user, exists := data.Users[userId]; exists {
		user.Email = email
		user.Password = string(hashedPassword)
		data.Users[userId] = user
		updatedUser = user
	}

	dbStructure := DBStructure{
		Chirps: data.Chirps,
		Users:  data.Users,
	}
	err = db.WriteDB(dbStructure)
	if err != nil {
		return User{}, err
	}

	return updatedUser, nil
}

func (db *DB) SaveRefreshToken(userId int, token string) error {
	data, err := db.LoadDB()
	if err != nil {
		return err
	}

	if user, exists := data.Users[userId]; exists {
		user.RefreshToken = token
		data.Users[userId] = user
	}

	dbStructure := DBStructure{
		Chirps: data.Chirps,
		Users:  data.Users,
	}
	err = db.WriteDB(dbStructure)
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) ValidateRefreshToken(token string) (User, error) {
	data, err := db.LoadDB()
	if err != nil {
		return User{}, err
	}

	var user User
	for _, u := range data.Users {
		if u.RefreshToken == token {
			user = u
		}
	}

	if user == (User{}) {
		return User{}, errors.New("refresh token not found")
	}

	return user, nil
}

func (db *DB) DeleteRefreshToken(user User) error {
	data, err := db.LoadDB()
	if err != nil {
		return err
	}

	if u, exist := data.Users[user.ID]; exist {
		u.RefreshToken = ""
		data.Users[user.ID] = u

		err = db.WriteDB(data)
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) UpgradeChirpy(userId int) error {
	// log.Printf("USER ID : %v", userId)
	data, err := db.LoadDB()
	if err != nil {
		return err
	}

	if u, exists := data.Users[userId]; exists {
		u.IsChirpyRed = true

		data.Users[userId] = u

		err = db.WriteDB(data)
		if err != nil {
			return err
		}
	} else {
		return errors.New("notfound")
	}

	return nil
}
