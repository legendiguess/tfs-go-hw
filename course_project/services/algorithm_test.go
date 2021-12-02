package services_test

import (
	"testing"

	"github.com/legendiguess/kraken-trade-bot/domain"
	"github.com/legendiguess/kraken-trade-bot/services"
)

type websocketClientServiceTest struct{}

func (websocketClientServiceTest *websocketClientServiceTest) GetTickerChannel() <-chan domain.Ticker {
	tickerChannel := make(chan domain.Ticker)

	ticker := domain.Ticker(make(map[string]interface{}))

	ticker["product_id"] = "test"
	ticker["ask"] = 100.0
	ticker["bid"] = 200.0

	tickerChannel <- ticker
	tickerChannel <- ticker
	close(tickerChannel)

	return tickerChannel
}

func TestAlgorithm(t *testing.T) {
	algorithm := services.NewAlgorithm(&websocketClientServiceTest{})

	algorithm.GetActionChannel()
}
