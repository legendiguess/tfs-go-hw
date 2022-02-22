package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/legendiguess/kraken-trade-bot/handlers"
	"github.com/legendiguess/kraken-trade-bot/services"
	"github.com/legendiguess/kraken-trade-bot/storage"
	log "github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	logger := log.New()
	logger.SetLevel(log.DebugLevel)

	credentials := storage.NewCredentialsStorage(logger)
	storage := storage.New(credentials, logger)

	userService := services.NewUsersService(storage)
	telegramBot := services.NewTelegramBot(userService, credentials, logger)

	instrumentSerivce := services.NewInstrumentService(storage)
	websocketClient := services.NewWebsocketClient(ctx, credentials, logger)
	handlers.NewServer(instrumentSerivce, websocketClient, logger)

	httpclient := services.NewHTTPClient(credentials)
	orderInfosService := services.NewOrderInfosService(storage)
	algorithm := services.NewAlgorithm(websocketClient)
	services.NewTradeBot(algorithm, instrumentSerivce, httpclient, orderInfosService, userService, telegramBot, logger)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done

	cancel()
}
