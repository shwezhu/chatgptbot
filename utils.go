package main

import (
	"errors"
	"fmt"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

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

func initSession(session *sessions.Session) {
	// session.Options.Path == "/"
	// MaxAge in seconds
	session.Options.MaxAge = 24 * 3600
	session.Values["authenticated"] = true
	session.Values["messages"] = []byte{}
}
