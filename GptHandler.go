package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/sessions"
	"github.com/sashabaranov/go-openai"
	"gopkg.in/boj/redistore.v1"
	"gorm.io/gorm"
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

		// get a session from store, whose key equals to session_id of the request's cookie
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

		messages, err := generateMessageFromRequest(r, session)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		balance, err := getBalance(db, session.Values["username"].(string))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
			return
		}

		messagesLength := calculateMessageLength(messages)
		if balance <= messagesLength {
			if _, err = fmt.Fprint(w, "you've reached limits"); err != nil {
				log.Println(err)
				return
			}
			return
		}

		resp, err := client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:    openai.GPT3Dot5Turbo,
				Messages: messages,
				// cannot be balance - messagesLength, think about if balance = 100000
				MaxTokens: 50,
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

		messages = append(
			messages,
			openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: resp.Choices[0].Message.Content,
			})
		err = saveSessionMessages(w, r, session, messages)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
			return
		}

		// 1 token ~= 4 chars in English
		balance = balance - (messagesLength-len(resp.Choices[0].Message.Content))/4
		err = updateTokens(db, session.Values["username"].(string), balance)
		if err != nil {
			log.Println(err)
			return
		}
	})
}

func generateMessageFromRequest(r *http.Request, session *sessions.Session) ([]openai.ChatCompletionMessage, error) {
	chatHistoryList, err := getSessionMessages(session)
	if err != nil {
		return nil, fmt.Errorf("failed to generate message: %v", err)
	}

	message, err := parseMessageFromRequest(r)
	if err != nil {
		return nil, fmt.Errorf("failed to generate message: %v", err)
	}

	chatHistoryList = append(
		chatHistoryList,
		openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: *message,
		},
	)

	return chatHistoryList, nil
}

// here should use encoding/gob for encoding and decoding, because this just communication between go program
// https://go.dev/blog/gob
// Internally, a slice is a pointer to an array, so passing a slice by value is very cheap.
// Returning a slice directly is easy to read compared with returning a pointer
func getSessionMessages(session *sessions.Session) ([]openai.ChatCompletionMessage, error) {
	var chatHistory []openai.ChatCompletionMessage

	if len(session.Values["messages"].([]byte)) == 0 {
		return chatHistory, nil
	}

	err := json.Unmarshal(session.Values["messages"].([]byte), &chatHistory)
	if err != nil {
		return nil, fmt.Errorf("failed to get session messages: %v", err)
	}
	return chatHistory, nil
}

// Internally, a slice is a pointer to an array, so passing a slice by value is very cheap.
func saveSessionMessages(w http.ResponseWriter, r *http.Request, session *sessions.Session, chatHistory []openai.ChatCompletionMessage) error {
	data, err := json.Marshal(chatHistory)
	if err != nil {
		return fmt.Errorf("failed to save session messages: %v", err)
	}

	session.Values["messages"] = data
	// set MaxAge whenever you call session.Save(r, w)
	// otherwise, MaxAge will be set back to default value
	session.Options.MaxAge = 24 * 3600
	if err = session.Save(r, w); err != nil {
		return fmt.Errorf("failed to save session messages: %v", err)
	}

	return nil
}

func parseMessageFromRequest(r *http.Request) (*string, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("failed to parse message: %v", err)
	}

	message := r.Form.Get("message")
	if message == "" {
		return nil, errors.New("failed to parse message: no message provided")
	}

	return &message, nil
}

func calculateMessageLength(messages []openai.ChatCompletionMessage) int {
	length := 0
	for _, v := range messages {
		length += len(v.Content)
	}
	return length
}

func getBalance(db *gorm.DB, username string) (int, error) {
	user := User{}

	// if no user found, return NoRecordFound error
	err := db.First(&user, "username = ?", username).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %v", err)
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
