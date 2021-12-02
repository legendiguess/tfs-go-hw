package services

import "github.com/legendiguess/kraken-trade-bot/domain"

type instrumentStorage interface {
	SaveInstrument(newInstrument domain.InstrumentConfig)
	GetInstrument() (domain.InstrumentConfig, bool)
}

type InstrumentService struct {
	storage instrumentStorage
}

func NewInstrumentService(storage instrumentStorage) *InstrumentService {
	return &InstrumentService{storage: storage}
}

func (instrumentService InstrumentService) SaveInstrument(newInstrument domain.InstrumentConfig) {
	instrumentService.storage.SaveInstrument(newInstrument)
}

func (instrumentService InstrumentService) GetInstrument() (domain.InstrumentConfig, bool) {
	return instrumentService.storage.GetInstrument()
}
