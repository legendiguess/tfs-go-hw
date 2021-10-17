package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	cookieAuth = "auth"
	userID     = "ID"
)

type Message struct {
	UserID string
	Text   string
}

type Messages []Message

func (messages *Messages) append(userID string, text string) {
	*messages = append(*messages, Message{UserID: userID, Text: text})
}

type GlobalChat struct {
	Messages Messages
}

type PrivateChat struct {
	FirstUserID  string
	SecondUserID string
	Messages     Messages
}

type PrivateChats []PrivateChat

func (privateChats *PrivateChats) findChat(firstUserID string, secondUserID string) (int, bool) {
	for index, chat := range *privateChats {
		if (chat.FirstUserID == firstUserID && chat.SecondUserID == secondUserID) || (chat.FirstUserID == secondUserID && chat.SecondUserID == firstUserID) {
			return index, true
		}
	}
	return 0, false
}

// Создает новый приватный чат с одним сообщением
func (privateChats *PrivateChats) append(firstUserID string, secondUserID string, message Message) {
	*privateChats = append(*privateChats, PrivateChat{FirstUserID: firstUserID, SecondUserID: secondUserID, Messages: Messages{}})
}

var storageUsers map[string]string
var globalChat GlobalChat
var privateChats PrivateChats

func main() {
	storageUsers = make(map[string]string)

	root := chi.NewRouter()
	root.Use(middleware.Logger)
	root.Post("/login", Login)
	root.Post("/registration", Registration)

	r := chi.NewRouter()
	r.Use(Auth)
	r.Get("/global", GlobalGetMessages)
	r.Post("/global", GlobalSendMessage)

	r.Get("/private/{userID}", PrivateGetMessages)
	r.Post("/private/{userID}", PrivateSendMessage)

	root.Mount("/", r)

	log.Fatal(http.ListenAndServe(":5000", root))
}

func PrivateGetMessages(w http.ResponseWriter, r *http.Request) {
	senderID, ok := r.Context().Value(userID).(string)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sendToID := chi.URLParam(r, "userID")
	if sendToID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, ok := storageUsers[sendToID]; !ok {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	chatID, ok := privateChats.findChat(senderID, sendToID)

	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	bytes, err := json.Marshal(privateChats[chatID].Messages)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(bytes)
}

func PrivateSendMessage(w http.ResponseWriter, r *http.Request) {
	senderID, ok := r.Context().Value(userID).(string)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sendToID := chi.URLParam(r, "userID")
	if sendToID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, ok := storageUsers[sendToID]; !ok {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	d, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var rawMessage RawMessage
	err = json.Unmarshal(d, &rawMessage)
	if err != nil || rawMessage.Text == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	chatID, ok := privateChats.findChat(senderID, sendToID)

	println(ok)
	if !ok {
		privateChats.append(senderID, sendToID, Message{UserID: senderID, Text: rawMessage.Text})
		return
	}

	privateChats[chatID].Messages.append(senderID, rawMessage.Text)
}

type RawMessage struct {
	Text string
}

func GlobalSendMessage(w http.ResponseWriter, r *http.Request) {
	id, ok := r.Context().Value(userID).(string)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	d, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var rawMessage RawMessage
	err = json.Unmarshal(d, &rawMessage)
	if err != nil || rawMessage.Text == "" {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		globalChat.Messages.append(id, rawMessage.Text)
	}
}

func GlobalGetMessages(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.Marshal(globalChat.Messages)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(bytes)
}

type User struct {
	Login string
}

func Registration(w http.ResponseWriter, r *http.Request) {
	d, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var user User
	err = json.Unmarshal(d, &user)
	if err != nil || user.Login == "" {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		if _, ok := storageUsers[user.Login]; ok { // Пользователь уже зарегистрирован
			w.WriteHeader(http.StatusConflict)
		} else {
			storageUsers[user.Login] = ""

			c := &http.Cookie{
				Name:  cookieAuth,
				Value: user.Login,
				Path:  "/",
			}
			http.SetCookie(w, c)
		}
	}
}

func Login(w http.ResponseWriter, r *http.Request) {
	d, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var user User
	err = json.Unmarshal(d, &user)
	if err != nil || user.Login == "" {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		if _, ok := storageUsers[user.Login]; ok {
			c := &http.Cookie{
				Name:  cookieAuth,
				Value: user.Login,
				Path:  "/",
			}
			http.SetCookie(w, c)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}
}

func Auth(handler http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieAuth)
		switch err {
		case nil:
		case http.ErrNoCookie:
			w.WriteHeader(http.StatusUnauthorized)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if c.Value == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		idCtx := context.WithValue(r.Context(), userID, c.Value)

		handler.ServeHTTP(w, r.WithContext(idCtx))
	}

	return http.HandlerFunc(fn)
}
