package main

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
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
