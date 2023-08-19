package handler

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/gorilla/sessions"
	"github.com/sashabaranov/go-openai"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/boj/redistore.v1"
	"gorm.io/gorm"
	. "gptbot/model"
	"log"
	"net/http"
)

func IndexHandler(store *redistore.RediStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session_id")
		if session.IsNew {
			session.Options.MaxAge = 30
			session.Values["authenticated"] = true
			session.Values["visit_time"] = 1
			_, _ = fmt.Fprintf(w, "welcome: %v", session.ID)
			_ = session.Save(r, w)
			return
		}

		session.Values["visit_time"] = session.Values["visit_time"].(int) + 1
		_ = session.Save(r, w)
		_, _ = fmt.Fprintf(w, "visit time: %v", session.Values["visit_time"])
	})
}

func LoginHandler(db *gorm.DB, store *redistore.RediStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not supported", http.StatusMethodNotAllowed)
			return
		}

		userInfo, err := parseUsernamePassword(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		user := User{}
		// if no user found, user = User{}
		err = db.Limit(1).Find(&user, "username = ?", (*userInfo)["username"]).Error
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
			return
		}

		if user == (User{}) {
			http.Error(w, "no such user, please register", http.StatusUnauthorized)
			return
		}

		// password is not correct
		if !checkPasswordHash((*userInfo)["password"], user.Password) {
			http.Error(w, "password is not correct", http.StatusUnauthorized)
			return
		}

		session, err := store.Get(r, "session_id")
		if !session.IsNew {
			_, err = fmt.Fprint(w, "you have logged in already")
			return
		}

		if err = initSession(session); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
			return
		}

		session.Values["username"] = (*userInfo)["username"]
		if err = session.Save(r, w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
		}

		if _, err = fmt.Fprint(w, "login successfully"); err != nil {
			log.Println(err)
			return
		}
	})
}

func LogoutHandler(store *redistore.RediStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get registers and returns a session for the given name and session store.
		session, err := store.Get(r, "session_id")
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to logout: %v", err), http.StatusInternalServerError)
			log.Printf("failed to logout: %v", err)
			return
		}

		if session.IsNew {
			if _, err := fmt.Fprint(w, "you have not logged in yet"); err != nil {
				log.Println(err)
			}
			return
		}

		// if you want to delete session: https://github.com/gorilla/sessions/issues/160
		session.Values["authenticated"] = false
		if err = session.Save(r, w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
			return
		}

		if _, err = fmt.Fprint(w, "logout successfully"); err != nil {
			log.Println(err)
			return
		}
	})
}

func RegisterHandler(db *gorm.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not supported", http.StatusMethodNotAllowed)
			return
		}

		userInfo, err := parseUsernamePassword(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		password, err := hashPassword((*userInfo)["password"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !db.Migrator().HasTable(&User{}) {
			// Migrate the schema - create table
			err = db.AutoMigrate(&User{})
			if err != nil {
				panic("failed to migrate the user schema")
			}
		}

		user := User{}
		err = db.Limit(1).Find(&user, "username = ?", (*userInfo)["username"]).Error
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
			return
		}

		// user exists
		if user != (User{}) {
			http.Error(w, "username has been taken", http.StatusConflict)
			return
		}

		// user doesn't exist, process registration
		user = User{Username: (*userInfo)["username"], Password: password}
		if err = db.Create(&user).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
		}

		if _, err = fmt.Fprint(w, "registered successfully"); err != nil {
			log.Println(err)
			return
		}
	})
}

func initSession(session *sessions.Session) error {
	// MaxAge in seconds
	session.Options.MaxAge = 24 * 3600
	session.Values["authenticated"] = true
	// if don't register, will have error:
	// gob: type not registered for interface: []openai.ChatCompletionMessage
	gob.Register([]openai.ChatCompletionMessage{})
	session.Values["messages"] = []openai.ChatCompletionMessage{}

	// session just need save once, otherwise client will receive two cookie
	// we will save session in LoginHandler function
	// err = session.Save(r, w)

	return nil
}

func parseUsernamePassword(r *http.Request) (*map[string]string, error) {
	userInfo := make(map[string]string)
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("failed to parse username and password: %v", err)
	}

	userInfo["username"] = r.Form.Get("username")
	userInfo["password"] = r.Form.Get("password")
	if userInfo["username"] == "" || userInfo["password"] == "" {
		return nil, errors.New("failed to parse username and password: no username or password")
	}

	return &userInfo, nil
}

// https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
// https://pkg.go.dev/golang.org/x/crypto/bcrypt
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}