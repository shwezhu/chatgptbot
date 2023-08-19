package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/sessions"
	"github.com/sashabaranov/go-openai"
	"gopkg.in/boj/redistore.v1"
	"gorm.io/gorm"
	. "gptbot/model"
	"log"
	"net/http"
	"os"
)

var client = openai.NewClient(os.Getenv("OPENAI_API_KEY"))

// Gpt3Dot5Handler https://github.com/sashabaranov/go-openai
func Gpt3Dot5Handler(db *gorm.DB, store *redistore.RediStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not supported", http.StatusMethodNotAllowed)
			return
		}

		session, err := store.Get(r, "session_id")
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get sessions: %v", err), http.StatusInternalServerError)
			log.Printf("failed to get sessions: %v", err)
			return
		}

		if session.IsNew || session.Values["authenticated"] == false {
			http.Error(w, "you have not logged in yet", http.StatusUnauthorized)
			return
		}

		message, err := parseMessage(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		updateMessage(session, message)

		messages := session.Values["messages"].([]openai.ChatCompletionMessage)
		balance, err := getTokens(db, session.Values["username"].(string))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
			return
		}

		historyMessageLength := calculateLength(&messages)
		if balance <= historyMessageLength {
			if _, err = fmt.Fprint(w, "you've reached limits"); err != nil {
				log.Println(err)
				return
			}
			return
		}

		resp, err := client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:     openai.GPT3Dot5Turbo,
				Messages:  messages,
				MaxTokens: balance - historyMessageLength,
			})

		if err != nil {
			http.Error(w, fmt.Sprintf("cannot respond: %v", err), http.StatusInternalServerError)
			log.Println(err)
			return
		}

		if _, err = fmt.Fprint(w, resp.Choices[0].Message.Content); err != nil {
			log.Println(err)
			return
		}

		updateMessage(session, &resp.Choices[0].Message.Content)
		if err = session.Save(r, w); err != nil {
			http.Error(w, fmt.Sprintf("failed to save session: %v", err), http.StatusUnauthorized)
			log.Printf("failed to save session: %v", err)
			return
		}

		// 1 token ~= 4 chars in English
		balance = balance - (historyMessageLength-len(resp.Choices[0].Message.Content))/4
		err = updateTokens(db, session.Values["username"].(string), balance)
		if err != nil {
			log.Println(err)
			return
		}
	})
}

func parseMessage(r *http.Request) (*string, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("failed to parse message: %v", err)
	}

	message := r.Form.Get("message")
	if message == "" {
		return nil, errors.New("failed to parse message: no message provided")
	}

	return &message, nil
}

func updateMessage(session *sessions.Session, message *string) {
	// Type assertions: https://go.dev/tour/methods/15
	session.Values["messages"] = append(
		session.Values["messages"].([]openai.ChatCompletionMessage),
		openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: *message,
		},
	)
}

func calculateLength(messages *[]openai.ChatCompletionMessage) int {
	length := 0
	for _, v := range *messages {
		length += len(v.Content)
	}
	return length
}

func getTokens(db *gorm.DB, username string) (int, error) {
	user := User{}
	// if no user found, return NoRecordFound error
	err := db.First(&user, "username = ?", username).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get token: %v", err)
	}

	return int(user.Tokens), nil
}

func updateTokens(db *gorm.DB, username string, tokens int) error {
	user := User{}
	// if no user found, return NoRecordFound error
	err := db.First(&user, "username = ?", username).Error
	if err != nil {
		return fmt.Errorf("failed to update tokens: %v", err)
	}

	user.Tokens = uint(tokens)
	if err = db.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to update tokens: %v", err)
	}

	return nil
}
