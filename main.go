package main

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gptbot/handler"
	"log"
	"net/http"
)

func main() {
	db, err := gorm.Open(sqlite.Open("gptbot.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	http.Handle("/", handler.IndexHandler())
	http.Handle("/login", handler.LoginHandler(db))
	http.Handle("/register", handler.RegisterHandler(db))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
