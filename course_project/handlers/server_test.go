package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/legendiguess/kraken-trade-bot/domain"
	"github.com/legendiguess/kraken-trade-bot/handlers"
	"github.com/stretchr/testify/assert"
)

type instrumentServiceTest struct{}

func (instrumentServiceTest *instrumentServiceTest) SaveInstrument(newInstrument domain.InstrumentConfig) {

}

func (instrumentServiceTest *instrumentServiceTest) GetInstrument() (domain.InstrumentConfig, bool) {
	return domain.InstrumentConfig{}, true
}

type websocketClientServiceTest struct{}

func (websocketClientServiceTest *websocketClientServiceTest) SubscribeToTicker(productIDs []string) {
}

func (websocketClientServiceTest *websocketClientServiceTest) UnsubscribeFromTicker(productIDs []string) {
}

type serverLoggerTest struct{}

func (serverLoggerTest *serverLoggerTest) Panic(args ...interface{}) {}

func TestInstrumentUpdate(t *testing.T) {
	handlers.NewServer(&instrumentServiceTest{}, &websocketClientServiceTest{}, &serverLoggerTest{})

	postBody, _ := json.Marshal(domain.InstrumentConfig{Symbol: "test_symbol"})

	newRequest, _ := http.NewRequest("PUT", "http://localhost:5000/instrument", bytes.NewBuffer(postBody))

	resp, err := http.DefaultClient.Do(newRequest)
	assert.Nil(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
