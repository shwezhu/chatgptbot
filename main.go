package main

import (
	"gopkg.in/boj/redistore.v1"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gptbot/handler"
	"log"
	"net/http"
	"os"
)

func main() {
	db, err := gorm.Open(sqlite.Open("gpt_bot.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}

	// https://github.com/gorilla/sessions
	// https://github.com/karankumarshreds/GoAuthentication/blob/master/readme.md
	var store *redistore.RediStore
	store, err = redistore.NewRediStore(10, "tcp", ":6379", "", []byte(os.Getenv("SESSION_KEY")))
	if err != nil {
		log.Fatal("failed to create Redis store")
	}

	http.HandleFunc("/", handler.IndexHandler)
	http.HandleFunc("/favicon.ico", handler.DoNothing)
	http.Handle("/login", handler.LoginHandler(db, store))
	http.Handle("/logout", handler.LogoutHandler(store))
	http.Handle("/register", handler.RegisterHandler(db))
	http.Handle("/chat/gpt-turbo", handler.Gpt3Dot5Handler(db, store))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
