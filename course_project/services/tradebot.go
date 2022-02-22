package services

import (
	"github.com/legendiguess/kraken-trade-bot/domain"
)

type algorithmService interface {
	GetActionChannel() <-chan domain.Action
}

type instrumentService interface {
	GetInstrument() (domain.InstrumentConfig, bool)
}

type httpClientService interface {
	Order(ticker string, side domain.OrderSide) (*domain.OrderInfo, error)
}

type orderInfosService interface {
	NewOrderInfo(orderInfo *domain.OrderInfo)
}

type telegramBotService interface {
	SendOrderInfo(chatID int64, orderInfo *domain.OrderInfo)
}

type tradeBotUsersStorage interface {
	GetUsers() []domain.User
}

type tradeBotLogger interface {
	Panicf(format string, args ...interface{})
	Printf(format string, args ...interface{})
}

type TradeBot struct {
}

func NewTradeBot(algorithmService algorithmService, instrumentService instrumentService, httpClientService httpClientService, orderInfosService orderInfosService, tradeBotUsersStorage tradeBotUsersStorage, telegramBot telegramBotService, tradeBotLogger tradeBotLogger) *TradeBot {
	tradeBot := TradeBot{}

	go func() {
		for action := range algorithmService.GetActionChannel() {
			if action == domain.ActionBuy || action == domain.ActionSell {
				side := domain.OrderSideBuy
				if action == domain.ActionSell {
					side = domain.OrderSideSell
				}

				instrument, _ := instrumentService.GetInstrument()

				orderInfo, err := httpClientService.Order(instrument.Symbol, side)
				if err != nil {
					tradeBotLogger.Panicf("%v", err)
				}
				tradeBotLogger.Printf("Successfully send %s %s order", side, instrument.Symbol)

				orderInfosService.NewOrderInfo(orderInfo)
				if err != nil {
					tradeBotLogger.Panicf("%v", err)
				}

				for _, user := range tradeBotUsersStorage.GetUsers() {
					telegramBot.SendOrderInfo(user.ChatID, orderInfo)
				}
			}
		}
	}()

	return &tradeBot
}
