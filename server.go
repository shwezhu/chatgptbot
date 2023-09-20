package main

import (
	"fmt"
	"github.com/gorilla/sessions"
	"gopkg.in/boj/redistore.v1"
	"gorm.io/gorm"
	"log"
	"net/http"
)

func newServer(db *gorm.DB, store *redistore.RediStore) *server {
	s := &server{
		db:     db,
		store:  store,
		router: http.NewServeMux(),
	}
	s.routes()
	return s
}

type server struct {
	db     *gorm.DB
	store  *redistore.RediStore
	router *http.ServeMux
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// ServeHTTP dispatches the request to the handler whose pattern most closely matches the request URL.
	s.router.ServeHTTP(w, r)
}

func (s *server) handleFavicon(_ http.ResponseWriter, _ *http.Request) {}

func (s *server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprint(w, "hello there")
}

// middleware
func (s *server) loggedInOnly(f func(http.ResponseWriter, *http.Request, *sessions.Session)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if user has logged in.
		session, err := s.store.Get(r, "session_id")
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to valid request:%v", err), http.StatusInternalServerError)
			log.Printf("failed to valid request: %v", err)
			return
		}
		if session.IsNew {
			_, err := fmt.Fprint(w, "you have not logged in yet")
			if err != nil {
				log.Println(err)
			}
			return
		}
		// Call the handler.
		f(w, r, session)
	}
}

func (s *server) postUsernameAndPasswordOnly(f func(w http.ResponseWriter, r *http.Request,
	username, password string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method is not supported", http.StatusMethodNotAllowed)
			return
		}
		// Parse username and password in form.
		username, password, err := getUsernameAndPassword(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		f(w, r, username, password)
	}
}

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

// findUser returns an empty User{} if user not found.
func (s *server) findUser(username string) (*User, error) {
	user := User{}
	return &user, s.db.Limit(1).Find(&user, "username = ?", username).Error
}

// validUserTable checks if user table exists, if not, create one.
func (s *server) validUserTable() error {
	if !s.db.Migrator().HasTable(&User{}) {
		// Migrate the schema - create table
		return s.db.AutoMigrate(&User{})
	}
	return nil
}
