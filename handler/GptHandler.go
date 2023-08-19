package handler

import (
	"context"
	"errors"
	"fmt"
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

		// 需要对 error 分类
		if err := updateMessage(w, r, store); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		session, err := store.Get(r, "session_id")
		if err != nil {
			http.Error(w, fmt.Sprintf("cannot get session: %v", err), http.StatusInternalServerError)
			return
		}

		messages := session.Values["messages"].([]openai.ChatCompletionMessage)
		balance, err := getTokens(db, session.Values["username"].(string))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
			return
		}

		messageTokens := calculateTokens(&messages)
		if balance <= messageTokens {
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
				MaxTokens: balance - messageTokens,
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

		balance = balance - messageTokens - len(resp.Choices[0].Message.Content)
		err = updateTokens(db, session.Values["username"].(string), balance)
		if err != nil {
			log.Println(err)
			return
		}
	})
}

func calculateTokens(messages *[]openai.ChatCompletionMessage) int {
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

func updateMessage(w http.ResponseWriter, r *http.Request, store *redistore.RediStore) error {
	session, err := store.Get(r, "session_id")
	if err != nil {
		return fmt.Errorf("cannot update message: %v", err)
	}

	if session.IsNew || session.Values["authenticated"] == false {
		return errors.New("you have not logged in yet")
	}

	message, err := parseMessage(r)
	if err != nil {
		return fmt.Errorf("failed to update message: %v", err)
	}

	// Type assertions: https://go.dev/tour/methods/15
	session.Values["messages"] = append(
		session.Values["messages"].([]openai.ChatCompletionMessage),
		openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: *message,
		},
	)

	if err = session.Save(r, w); err != nil {
		return fmt.Errorf("cannot update message: %v", err)
	}

	return nil
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
