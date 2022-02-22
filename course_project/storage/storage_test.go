package storage

import (
	"testing"

	"github.com/legendiguess/kraken-trade-bot/domain"
	"github.com/stretchr/testify/assert"
)

type databaseCredentials struct {
}

func (databaseCredentials *databaseCredentials) GetDatabaseDSN() string {
	return `user=user password=user dbname=kraken-trade-bot-test port=5432 TimeZone=Europe/Moscow`
}

type databaseLogger struct{}

func (databaseLogger *databaseLogger) Panicf(format string, args ...interface{}) {

}

func newTestStorage() *Storage {
	storage := New(&databaseCredentials{}, &databaseLogger{})
	storage.dataBase.Migrator().DropTable(&domain.InstrumentConfig{}, &domain.User{})
	storage.dataBase.AutoMigrate(&domain.InstrumentConfig{}, &domain.User{})
	return storage
}

func TestSaveAndGetInstrument(t *testing.T) {
	testStoage := newTestStorage()

	_, ok := testStoage.GetInstrument()

	testStoage.SaveInstrument(domain.InstrumentConfig{Symbol: "test1"})

	testInstrument := domain.InstrumentConfig{}
	testInstrument.Symbol = "test2"

	testStoage.SaveInstrument(testInstrument)

	instrument, ok := testStoage.GetInstrument()

	assert.Equal(t, true, ok)

	assert.Equal(t, testInstrument, instrument)
}

func TestUsers(t *testing.T) {
	testStoage := newTestStorage()

	assert.Equal(t, []domain.User{}, testStoage.GetUsers())

	user1 := domain.User{ChatID: 1}
	user2 := domain.User{ChatID: 1}

	testStoage.NewUser(&user1)
	testStoage.NewUser(&user2)

	assert.Equal(t, []domain.User{user1, user1}, testStoage.GetUsers())

	findedUser, ok := testStoage.FindUser(&user2)

	assert.Equal(t, true, ok)
	assert.Equal(t, user2, findedUser)
}
