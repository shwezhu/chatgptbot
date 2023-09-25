package main

import (
	"github.com/sashabaranov/go-openai"
	"gopkg.in/boj/redistore.v1"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestHandleRegister(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("gpt_bot.db"), &gorm.Config{})
	if err != nil {
		t.Fatal("failed to connect database")
	}
	srv := newServer()
	srv.db = db
	tests := []struct {
		name   string
		param  string
		expect string
	}{
		// Use x-www-form-urlencoded format, not json format: {"username":"david", "password":"778899"}
		{"base case - 1", "username=david&password=778899123", "registered successfully\n"},
		{"base case - 2", "username=david&password=my_password", "username has been taken\n"},
		{"bad case", "", "failed to parse username and password: no username or password\n"},
	}
	for _, tt := range tests {
		// This is a form, not json data
		urlEncodedForm := strings.NewReader(tt.param)
		r, err := http.NewRequest(http.MethodPost, "/register", urlEncodedForm)
		if err != nil {
			t.Fatal(err)
		}
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		if string(w.Body.Bytes()) != tt.expect {
			t.Errorf("expected: %v, got: %v", tt.expect, string(w.Body.Bytes()))
		}
	}
}

func TestHandleGpt3Dot5Turbo(t *testing.T) {
	// Init settings.
	db, err := gorm.Open(sqlite.Open("gpt_bot.db"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	store, err := redistore.NewRediStore(10, "tcp", ":6379", "", []byte(os.Getenv("SESSION_KEY")))
	if err != nil {
		t.Fatalf("failed to create redis store: %v", err)
	}
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	srv := newServer()
	srv.db = db
	srv.store = store
	srv.client = client
	// Login
	urlEncodedForm := strings.NewReader("username=david&password=778899a")
	r, err := http.NewRequest(http.MethodPost, "/login", urlEncodedForm)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatal("w.Code != http.StatusOK")
	}
	// chat with gpt  round - 1
	message := strings.NewReader("message=who is the president of America")
	r, err = http.NewRequest(http.MethodPost, "/chat/gpt-3-turbo", message)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	cookie := w.Header().Get("Set-Cookie")
	r.Header.Set("Cookie", cookie)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatal("w.Code != http.StatusOK")
	}
	t.Log(w.Body)
	// chat with gpt  round - 2
	message = strings.NewReader("message=tell me more about him")
	r, err = http.NewRequest(http.MethodPost, "/chat/gpt-3-turbo", message)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Cookie", cookie)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatal("w.Code != http.StatusOK")
	}
	t.Log(w.Body)
}
