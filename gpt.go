package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/sessions"
	"github.com/sashabaranov/go-openai"
	"log"
	"net/http"
)

func (s *server) handleGpt3Dot5Turbo(w http.ResponseWriter, r *http.Request,
	session *sessions.Session) {
	// Combine message from request and history message into a slice for later use.
	messages, err := formatMessages(r, session)
	// Get balance tokens of the user.
	balance, err := s.getUserTokens(session.Values["username"].(string))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	// Compare balance with the length of messages which will sent to openai.
	messagesLength := calculateTokens(messages)
	if balance <= messagesLength {
		if _, err = fmt.Fprint(w, "you've reached limits"); err != nil {
			log.Println(err)
			return
		}
		return
	}
	// Send message to openai.
	resp, err := s.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo,
			Messages: messages,
			// cannot be 'balance - messagesLength' here,
			// think about if balance = 100000
			MaxTokens: 50})
	if err != nil {
		http.Error(w, fmt.Sprintf("cannot respond: %v", err), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	// Send response to user.
	if _, err = fmt.Fprint(w, resp.Choices[0].Message.Content); err != nil {
		log.Println(err)
		return
	}
	// Append response to 'messages' and  Save chat history into session.
	messages = append(
		messages,
		openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: resp.Choices[0].Message.Content,
		})
	err = saveChatHistory(w, r, session, messages)
	if err != nil {
		log.Println(err)
		return
	}
	// Update the balance of user.
	balance = balance - calculateTokens(messages)
	err = s.updateUserTokens(session.Values["username"].(string), balance)
	if err != nil {
		log.Println(err)
		return
	}
}

func (s *server) getUserTokens(username string) (int, error) {
	user, err := s.findUser(username)
	if err != nil {
		return 0, fmt.Errorf("failed to get user tokens: %v", err)
	}
	return int(user.Tokens), nil
}

func (s *server) updateUserTokens(username string, tokens int) error {
	user, err := s.findUser(username)
	if err != nil {
		return fmt.Errorf("failed to update tokens: %v", err)
	}
	user.Tokens = uint(tokens)
	err = s.db.Save(&user).Error
	if err != nil {
		return fmt.Errorf("failed to update user tokens: %v", err)
	}
	return nil
}

// formatMessages gets history message from session and current message from request,
//
// then combines them into a slice which achieves the history message feature.
func formatMessages(r *http.Request, session *sessions.Session) ([]openai.ChatCompletionMessage, error) {
	// Get chat history stored in session.
	history, err := getHistoryMessages(session)
	if err != nil {
		return nil, fmt.Errorf("failed to format messages: %v", err)
	}
	// Parse message from request.
	message, err := parseMessageFromRequest(r)
	if err != nil {
		return nil, fmt.Errorf("failed to format messages: %v", err)
	}
	// Combine the message parsed from request into history and send it to openai later.
	history = append(
		history,
		openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: message,
		},
	)
	return history, nil
}

// getHistoryMessages returns a slice of openai.ChatCompletionMessage
func getHistoryMessages(session *sessions.Session) ([]openai.ChatCompletionMessage, error) {
	var history []openai.ChatCompletionMessage
	// No history message stored in session.
	if len(session.Values["messages"].([]byte)) == 0 {
		return history, nil
	}
	// Decode history messages stored in session.
	err := json.Unmarshal(session.Values["messages"].([]byte), &history)
	if err != nil {
		return nil, fmt.Errorf("failed to get history messages: %v", err)
	}
	return history, nil
}

// parseMessageFromRequest returns a message parsed from http request.
// If message is empty, returns an error.
func parseMessageFromRequest(r *http.Request) (string, error) {
	if err := r.ParseForm(); err != nil {
		return "", fmt.Errorf("failed to parse message: %v", err)
	}
	message := r.Form.Get("message")
	if message == "" {
		return "", errors.New("failed to parse message: no message provided in the request")
	}
	return message, nil
}

// 1 token ~= 4 characters in English
func calculateTokens(messages []openai.ChatCompletionMessage) int {
	length := 0
	for _, v := range messages {
		length += len(v.Content)
	}
	return length / 4
}

// saveChatHistory saves chat history into session.
func saveChatHistory(w http.ResponseWriter, r *http.Request,
	session *sessions.Session, chatHistory []openai.ChatCompletionMessage) error {
	data, err := json.Marshal(chatHistory)
	if err != nil {
		return fmt.Errorf("failed to save session messages: %v", err)
	}
	session.Values["messages"] = data
	// set MaxAge whenever before you call session.Save(r, w)
	// otherwise, MaxAge will be set back to default value.
	// this is a bug of gorilla/session
	session.Options.MaxAge = 24 * 3600
	if err = session.Save(r, w); err != nil {
		return fmt.Errorf("failed to save session messages: %v", err)
	}
	return nil
}
