package handler

import (
	"gorm.io/gorm"
	"net/http"
)

// https://github.com/sashabaranov/go-openai
func Gpt3DotHandler(db *gorm.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	})
}
