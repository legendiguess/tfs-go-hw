package services_test

import (
	"testing"

	"github.com/legendiguess/kraken-trade-bot/domain"
	"github.com/legendiguess/kraken-trade-bot/services"
	"github.com/stretchr/testify/assert"
)

type testUsersStorage struct {
	users []domain.User
}

func (testUsersStorage *testUsersStorage) NewUser(newUser *domain.User) {
	testUsersStorage.users = append(testUsersStorage.users, *newUser)
}

func (testUsersStorage *testUsersStorage) GetUsers() []domain.User {
	return testUsersStorage.users
}

func (testUsersStorage *testUsersStorage) FindUser(findUser *domain.User) (domain.User, bool) {
	for _, user := range testUsersStorage.users {
		if user.ChatID == findUser.ChatID {
			return user, true
		}
	}

	return domain.User{}, false
}

func TestCheckAddUser(t *testing.T) {
	testUsersStorage := testUsersStorage{}

	userService := services.NewUsersService(&testUsersStorage)

	assert.Equal(t, []domain.User(nil), userService.GetUsers())

	user1 := domain.User{ChatID: 1}

	userService.CheckAddUser(&user1)

	assert.Equal(t, []domain.User{user1}, userService.GetUsers())

	userService.CheckAddUser(&user1)
	userService.CheckAddUser(&user1)

	assert.Equal(t, []domain.User{user1}, userService.GetUsers())

	user2 := domain.User{ChatID: 2}

	userService.CheckAddUser(&user2)

	assert.Equal(t, []domain.User{user1, user2}, userService.GetUsers())
}
