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
	mutex    sync.Mutex
}

func (globalChat *GlobalChat) getMessages() Messages {
	globalChat.mutex.Lock()
	defer globalChat.mutex.Unlock()

	copyOfArray := make(Messages, len(globalChat.Messages))
	copy(copyOfArray, globalChat.Messages)

	return copyOfArray
}

func (globalChat *GlobalChat) newMessage(userID string, text string) {
	globalChat.mutex.Lock()
	defer globalChat.mutex.Unlock()

	globalChat.Messages.append(userID, text)
}

type PrivateChat struct {
	FirstUserID  string
	SecondUserID string
	Messages     Messages
}

type PrivateChats struct {
	storage []PrivateChat
	mutex   sync.Mutex
}

func (privateChats *PrivateChats) findChat(firstUserID string, secondUserID string) (int, bool) {
	privateChats.mutex.Lock()
	defer privateChats.mutex.Unlock()

	for index, chat := range privateChats.storage {
		if (chat.FirstUserID == firstUserID && chat.SecondUserID == secondUserID) || (chat.FirstUserID == secondUserID && chat.SecondUserID == firstUserID) {
			return index, true
		}
	}
	return 0, false
}

// Создает новый приватный чат и возвращает его индекс
func (privateChats *PrivateChats) newChat(firstUserID string, secondUserID string) int {
	privateChats.mutex.Lock()
	defer privateChats.mutex.Unlock()

	privateChats.storage = append(privateChats.storage, PrivateChat{FirstUserID: firstUserID, SecondUserID: secondUserID, Messages: Messages{}})

	return len(privateChats.storage) - 1
}

func (privateChats *PrivateChats) getMessages(chatIndex int) Messages {
	privateChats.mutex.Lock()
	defer privateChats.mutex.Unlock()

	copyOfArray := make(Messages, len(privateChats.storage[chatIndex].Messages))
	copy(copyOfArray, privateChats.storage[chatIndex].Messages)

	return copyOfArray

}

func (privateChats *PrivateChats) newMessage(chatIndex int, senderID string, text string) {
	privateChats.mutex.Lock()
	defer privateChats.mutex.Unlock()

	privateChats.storage[chatIndex].Messages.append(senderID, text)
}

type Users struct {
	mutex   sync.Mutex
	storage map[string]string
}

func (users *Users) checkIsUserExist(userID string) bool {
	users.mutex.Lock()
	defer users.mutex.Unlock()

	_, ok := users.storage[userID]

	return ok
}

func (users *Users) newUser(userID string) {
	users.mutex.Lock()
	defer users.mutex.Unlock()

	users.storage[userID] = ""
}

type System struct {
	users        Users
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

	is_user_exists := system.users.checkIsUserExist(sendToID)

	if !is_user_exists {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	chatID, ok := system.privateChats.findChat(senderID, sendToID)

	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	bytes, err := json.Marshal(system.privateChats.getMessages(chatID))
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

	is_user_exists := system.users.checkIsUserExist(sendToID)

	if !is_user_exists {
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

	chatIndex, ok := system.privateChats.findChat(senderID, sendToID)

	if !ok {
		chatIndex = system.privateChats.newChat(senderID, sendToID)
		return
	}

	system.privateChats.newMessage(chatIndex, senderID, rawMessage.Text)
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
		system.globalChat.newMessage(id, rawMessage.Text)
	}
}

func (system *System) GlobalGetMessages(w http.ResponseWriter, r *http.Request) {
	messages := system.globalChat.getMessages()

	bytes, err := json.Marshal(messages)

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
		is_user_exists := system.users.checkIsUserExist(user.Login)

		if is_user_exists {
			w.WriteHeader(http.StatusConflict)
		} else {
			system.users.newUser(user.Login)

			c := &http.Cookie{
				Name:  cookieAuth,
				Value: user.Login,
				Path:  "/",
			}
			http.SetCookie(w, c)
		}
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
		is_user_exists := system.users.checkIsUserExist(user.Login)

		if is_user_exists {
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
		users:        Users{},
		globalChat:   GlobalChat{},
		privateChats: PrivateChats{},
	}

	system.users.storage = make(map[string]string)

	root := chi.NewRouter()
	root.Mount("/", system.Routes())

	log.Fatal(http.ListenAndServe(":5000", root))
}
