package main

import (
	"errors"
	"fmt"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
)

func (s *server) handleAuthLogin(w http.ResponseWriter, r *http.Request,
	username, password string) {
	// Query user by username in database.
	user, err := s.findUser(username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	if *user == (User{}) {
		http.Error(w, "you have not registered yet", http.StatusUnauthorized)
		return
	}
	// Compare provided password with password stored in database.
	if !comparePasswordHash(user.Password, password) {
		http.Error(w, "password is incorrect", http.StatusUnauthorized)
		return
	}
	// Get session from store.
	session, err := s.store.Get(r, "session_id")
	if !session.IsNew {
		_, err = fmt.Fprint(w, "you have logged in already")
		return
	}
	// Session is new, config session.
	session.Options.MaxAge = 24 * 3600 // MaxAge in seconds
	session.Values["authenticated"] = true
	session.Values["messages"] = []byte{}
	session.Values["username"] = username
	// Save session.
	if err = session.Save(r, w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	if _, err = fmt.Fprint(w, "login successfully"); err != nil {
		log.Println(err)
		return
	}
}

func (s *server) handleAuthLogout(w http.ResponseWriter, r *http.Request,
	session *sessions.Session) {
	// delete session from session store
	// https://github.com/gorilla/sessions/issues/160
	session.Options.MaxAge = -1
	session.Values["authenticated"] = false
	if err := session.Save(r, w); err != nil {
		http.Error(w, fmt.Sprintf("failed to log out:%v", err), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	if _, err := fmt.Fprint(w, "log out successfully"); err != nil {
		log.Println(err)
		return
	}
}

func (s *server) handleRegister(w http.ResponseWriter, _ *http.Request,
	username, password string) {
	// Store encrypted password in database.
	hashedPassword, err := hashPassword(password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Check if the user table exists, if not, create one.
	err = s.validUserTable()
	if err != nil {
		log.Fatal("failed to migrate the user schema")
	}
	// Check if user has existed in the database.
	user, err := s.findUser(username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	if *user != (User{}) {
		http.Error(w, "username has been taken", http.StatusConflict)
		return
	}
	user.Username = username
	user.Password = hashedPassword
	// Save user into database.
	if err = s.db.Create(user).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
	}
	if _, err = fmt.Fprint(w, "registered successfully"); err != nil {
		log.Println(err)
		return
	}
}

// validUserTable checks if user table exists, if not, create one.
func (s *server) validUserTable() error {
	if !s.db.Migrator().HasTable(&User{}) {
		// Migrate the schema - create table
		return s.db.AutoMigrate(&User{})
	}
	return nil
}

func getUsernameAndPassword(r *http.Request) (username string, password string, err error) {
	if e := r.ParseForm(); err != nil {
		err = fmt.Errorf("failed to parse username and password: %v", e)
		return
	}
	username = r.Form.Get("username")
	password = r.Form.Get("password")
	if username == "" || password == "" {
		err = errors.New("failed to parse username and password: no username or password")
		return
	}
	return
}

func comparePasswordHash(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
// https://pkg.go.dev/golang.org/x/crypto/bcrypt
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}
