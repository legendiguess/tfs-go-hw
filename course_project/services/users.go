package services

import (
	"github.com/legendiguess/kraken-trade-bot/domain"
)

type usersStorage interface {
	NewUser(newUser *domain.User)
	GetUsers() []domain.User
	FindUser(findUser *domain.User) (domain.User, bool)
}

func NewUsersService(storage usersStorage) *UsersService {
	return &UsersService{storage: storage}
}

type UsersService struct {
	storage usersStorage
}

// Save user to the database, if a user already exists function does nothing
func (usersService *UsersService) CheckAddUser(user *domain.User) {
	_, ok := usersService.storage.FindUser(user)

	if !ok {
		usersService.storage.NewUser(user)
	}
}

func (usersService *UsersService) GetUsers() []domain.User {
	return usersService.storage.GetUsers()
}
