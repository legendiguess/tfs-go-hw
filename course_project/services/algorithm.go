package services

import (
	"github.com/legendiguess/kraken-trade-bot/domain"
)

type Algorithm struct {
	lastAction          domain.Action
	previousActionPrice float64
	instrument          domain.InstrumentConfig
	actionChannel       <-chan domain.Action
}

type websocketClientService interface {
	GetTickerChannel() <-chan domain.Ticker
}

func NewAlgorithm(websocketClientService websocketClientService) *Algorithm {
	algorithm := Algorithm{}
	actionChannel := make(chan domain.Action)

	go func() {
		defer close(actionChannel)
		for ticker := range websocketClientService.GetTickerChannel() {
			action := domain.ActionNothing

			tickerSymbol := ticker.GetSymbol()
			if algorithm.instrument.Symbol != tickerSymbol {
				algorithm.instrument.Symbol = tickerSymbol
				algorithm.lastAction = domain.ActionSell
				algorithm.previousActionPrice = 0.0
				actionChannel <- action
				continue
			}

			ask := ticker.GetAsk()
			bid := ticker.GetBid()

			if algorithm.lastAction == domain.ActionSell {
				algorithm.previousActionPrice = ask
				action = domain.ActionBuy
			} else if algorithm.lastAction == domain.ActionBuy {
				if algorithm.previousActionPrice-bid <= algorithm.previousActionPrice/1000*-1 {
					algorithm.previousActionPrice = bid
					action = domain.ActionSell
				}
			}
			if action != domain.ActionNothing {
				algorithm.lastAction = action
			}

			actionChannel <- action
		}
	}()

	algorithm.actionChannel = actionChannel
	return &algorithm
}

func (alogrithm *Algorithm) GetActionChannel() <-chan domain.Action {
	return alogrithm.actionChannel
}
