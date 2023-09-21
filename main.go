package main

import (
	"github.com/sashabaranov/go-openai"
	"gopkg.in/boj/redistore.v1"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"net/http"
	"os"
)

func main() {
	db, err := gorm.Open(sqlite.Open("gpt_bot.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}
	store, err := redistore.NewRediStore(10, "tcp", ":6379", "", []byte(os.Getenv("SESSION_KEY")))
	if err != nil {
		log.Fatal("failed to create Redis store")
	}
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	s := newServer(db, store, client)
	log.Fatal(http.ListenAndServe(":8080", s))
}
