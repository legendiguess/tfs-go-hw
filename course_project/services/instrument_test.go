package services_test

import (
	"testing"

	"github.com/legendiguess/kraken-trade-bot/domain"
	"github.com/legendiguess/kraken-trade-bot/services"
	"github.com/stretchr/testify/assert"
)

type instrumentStorageTest struct {
	instrument domain.InstrumentConfig
}

func (instrumentStorageTest *instrumentStorageTest) SaveInstrument(newInstrument domain.InstrumentConfig) {
	instrumentStorageTest.instrument = newInstrument
}

func (instrumentStorageTest *instrumentStorageTest) GetInstrument() (domain.InstrumentConfig, bool) {
	return instrumentStorageTest.instrument, true
}

func TestInstrumentService(t *testing.T) {
	instrumentService := services.NewInstrumentService(&instrumentStorageTest{})

	testInstrument := domain.InstrumentConfig{Symbol: "test"}
	instrumentService.SaveInstrument(testInstrument)

	instrument, ok := instrumentService.GetInstrument()

	assert.Equal(t, true, ok)
	assert.Equal(t, testInstrument, instrument)
}
