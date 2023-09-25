package main

import (
	"errors"
	"fmt"
	"github.com/gorilla/sessions"
	"github.com/sashabaranov/go-openai"
	"gopkg.in/boj/redistore.v1"
	"gorm.io/gorm"
	"log"
	"net/http"
)

// Don't set dependencies here, for set dependencies later for easy test.
func newServer() *server {
	s := &server{
		router: http.NewServeMux(),
	}
	s.routes()
	return s
}

type server struct {
	db     *gorm.DB
	store  *redistore.RediStore
	client *openai.Client
	router *http.ServeMux
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// ServeHTTP dispatches the request to the handler whose pattern most closely matches the request URL.
	s.router.ServeHTTP(w, r)
}

// handleFavicon handles request "/favicon.ico", otherwise the handler of pattern "/" will be "called" twice.
// https://stackoverflow.com/a/57682227/16317008
func (s *server) handleFavicon(_ http.ResponseWriter, _ *http.Request) {}

func (s *server) handleGreeting(w http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprint(w, "hello there")
}

// middleware
// act as a filter which only allow the logged in request to pass
func (s *server) authenticatedOnly(f func(http.ResponseWriter, *http.Request, *sessions.Session)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if user has logged in.
		session, err := s.store.Get(r, "session_id")
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to valid request:%v", err), http.StatusInternalServerError)
			log.Printf("failed to valid request: %v", err)
			return
		}
		if session.IsNew || session.Values["authenticated"] == false {
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

// middleware
// only allow POST method and has password and username parameter to pass
func (s *server) postUsernamePasswordOnly(f func(w http.ResponseWriter, r *http.Request,
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

// findUser returns an empty User{} if user not found.
func (s *server) findUser(username string) (*User, error) {
	user := User{}
	// s.db.Limit(1).Find(): returns an empty User{} if user not found.
	err := s.db.Limit(1).Find(&user, "username = ?", username).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %v", err)
	}
	if user == (User{}) {
		return nil, errors.New("failed to find user: user doesn't exist")
	}
	return &user, nil
}
