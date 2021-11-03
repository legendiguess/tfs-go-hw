package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

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

type System struct {
	mutex        sync.Mutex
	storageUsers map[string]string
	globalChat   GlobalChat
	privateChats PrivateChats
}

func (system *System) PrivateGetMessages(w http.ResponseWriter, r *http.Request) {
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
	system.mutex.Lock()
	defer system.mutex.Unlock()
	if _, ok := system.storageUsers[sendToID]; !ok {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	chatID, ok := system.privateChats.findChat(senderID, sendToID)

	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	bytes, err := json.Marshal(system.privateChats[chatID].Messages)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(bytes)
}

func (system *System) PrivateSendMessage(w http.ResponseWriter, r *http.Request) {
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
	system.mutex.Lock()
	if _, ok := system.storageUsers[sendToID]; !ok {
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

	chatID, ok := system.privateChats.findChat(senderID, sendToID)

	if !ok {
		system.privateChats.append(senderID, sendToID, Message{UserID: senderID, Text: rawMessage.Text})
		return
	}

	system.privateChats[chatID].Messages.append(senderID, rawMessage.Text)
	system.mutex.Unlock()
}

type RawMessage struct {
	Text string
}

func (system *System) GlobalSendMessage(w http.ResponseWriter, r *http.Request) {
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
		system.mutex.Lock()
		system.globalChat.Messages.append(id, rawMessage.Text)
		system.mutex.Unlock()
	}
}

func (system *System) GlobalGetMessages(w http.ResponseWriter, r *http.Request) {
	system.mutex.Lock()
	bytes, err := json.Marshal(system.globalChat.Messages)
	system.mutex.Unlock()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(bytes)
}

type User struct {
	Login string
}

func (system *System) Registration(w http.ResponseWriter, r *http.Request) {
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
		system.mutex.Lock()
		if _, ok := system.storageUsers[user.Login]; ok { // Пользователь уже зарегистрирован
			w.WriteHeader(http.StatusConflict)
		} else {
			system.storageUsers[user.Login] = ""

			c := &http.Cookie{
				Name:  cookieAuth,
				Value: user.Login,
				Path:  "/",
			}
			http.SetCookie(w, c)
		}
		system.mutex.Unlock()
	}
}

func (system *System) Login(w http.ResponseWriter, r *http.Request) {
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
		system.mutex.Lock()
		if _, ok := system.storageUsers[user.Login]; ok {
			c := &http.Cookie{
				Name:  cookieAuth,
				Value: user.Login,
				Path:  "/",
			}
			http.SetCookie(w, c)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
		system.mutex.Unlock()
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

func (system *System) Routes() chi.Router {
	root := chi.NewRouter()
	root.Use(middleware.Logger)
	root.Post("/login", system.Login)
	root.Post("/registration", system.Registration)

	r := chi.NewRouter()
	r.Use(Auth)
	r.Get("/global", system.GlobalGetMessages)
	r.Post("/global", system.GlobalSendMessage)

	r.Get("/private/{userID}", system.PrivateGetMessages)
	r.Post("/private/{userID}", system.PrivateSendMessage)

	root.Mount("/", r)

	return root
}

func main() {
	system := System{
		mutex:        sync.Mutex{},
		storageUsers: make(map[string]string),
		globalChat:   GlobalChat{},
		privateChats: PrivateChats{},
	}

	root := chi.NewRouter()
	root.Mount("/", system.Routes())

	log.Fatal(http.ListenAndServe(":5000", root))
}
