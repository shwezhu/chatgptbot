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
func (s *server) validateRequest(f func(http.ResponseWriter, *http.Request, *sessions.Session)) http.HandlerFunc {
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

func (s *server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {

}

func (s *server) handleAuthLogout(w http.ResponseWriter, r *http.Request, session *sessions.Session) {
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
