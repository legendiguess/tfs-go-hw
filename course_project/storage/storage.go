package storage

import (
	"errors"

	"github.com/legendiguess/kraken-trade-bot/domain"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type databaseDSNStorage interface {
	GetDatabaseDSN() string
}

type storageLogger interface {
	Panicf(format string, args ...interface{})
}

type Storage struct {
	dataBase *gorm.DB
	logger   storageLogger
}

func New(databaseDSNStorage databaseDSNStorage, storageLogger storageLogger) *Storage {
	dataBase, err := gorm.Open(postgres.New(
		postgres.Config{
			DSN:                  databaseDSNStorage.GetDatabaseDSN(),
			PreferSimpleProtocol: true,
		}), &gorm.Config{})

	if err != nil {
		storageLogger.Panicf("%v", err)
	}

	storage := Storage{dataBase: dataBase, logger: storageLogger}
	storage.dataBase.AutoMigrate(&domain.OrderInfo{}, &domain.User{}, &domain.InstrumentConfig{})

	return &storage
}

func (storage *Storage) NewOrderInfo(orderInfo *domain.OrderInfo) {
	result := storage.dataBase.Create(orderInfo)

	if result.Error != nil {
		storage.logger.Panicf("%v", result.Error)
	}
}

func (storage *Storage) NewUser(newUser *domain.User) {
	result := storage.dataBase.Create(&newUser)

	if result.Error != nil {
		storage.logger.Panicf("%v", result.Error)
	}
}

func (storage *Storage) FindUser(findUser *domain.User) (domain.User, bool) {
	var user domain.User

	result := storage.dataBase.Where(user).Take(&user)

	isFound := !errors.Is(result.Error, gorm.ErrRecordNotFound)
	if isFound && result.Error != nil {
		storage.logger.Panicf("%v", result.Error)
	}

	return user, isFound
}

func (storage *Storage) GetUsers() []domain.User {
	var users []domain.User

	result := storage.dataBase.Find(&users)

	if result.Error != nil {
		storage.logger.Panicf("%v", result.Error)
	}

	return users
}

// Save instrument to the database
func (storage *Storage) SaveInstrument(newInstrument domain.InstrumentConfig) {
	var instrument domain.InstrumentConfig
	result := storage.dataBase.Take(&instrument)

	isFound := !errors.Is(result.Error, gorm.ErrRecordNotFound)
	if isFound && result.Error != nil {
		storage.logger.Panicf("%v", result.Error)
	}

	if isFound {
		storage.dataBase.Where("Symbol = ?", instrument.Symbol).Delete(&instrument)
	}

	storage.dataBase.Create(&newInstrument)
}

func (storage *Storage) GetInstrument() (domain.InstrumentConfig, bool) {
	var instrument domain.InstrumentConfig

	result := storage.dataBase.Take(&instrument)

	isFound := !errors.Is(result.Error, gorm.ErrRecordNotFound)

	if isFound && result.Error != nil {
		storage.logger.Panicf("%v", result.Error)
	}

	return instrument, isFound
}
