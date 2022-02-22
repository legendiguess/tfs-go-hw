package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/legendiguess/kraken-trade-bot/domain"
)

type usersService interface {
	CheckAddUser(user *domain.User)
	GetUsers() []domain.User
}

type telegramBotCredentials interface {
	GetTelegramBotAPIToken() string
}

type telegramBotLogger interface {
	Panic(args ...interface{})
	Panicf(format string, args ...interface{})
}

type TelegramBot struct {
	bot          *tgbotapi.BotAPI
	usersService usersService
	logger       telegramBotLogger
}

func NewTelegramBot(usersService usersService, telegramBotCredentials telegramBotCredentials, telegramBotLogger telegramBotLogger) *TelegramBot {
	telegramBot := TelegramBot{usersService: usersService, logger: telegramBotLogger}

	var err error

	telegramBot.bot, err = tgbotapi.NewBotAPI(telegramBotCredentials.GetTelegramBotAPIToken())
	if err != nil {
		telegramBot.logger.Panic(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 10

	updates := telegramBot.bot.GetUpdatesChan(u)

	go func() {
		for update := range updates {
			if update.Message == nil {
				continue
			}

			if update.Message.Text == "/start" {
				telegramBot.usersService.CheckAddUser(&domain.User{ChatID: update.Message.Chat.ID})
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Вы подписались на получение информации по ордерам 👍")
				telegramBot.bot.Send(msg)
			}
		}
	}()

	return &telegramBot
}

func (telegramBot *TelegramBot) SendOrderInfo(chatID int64, orderInfo *domain.OrderInfo) {
	template := "%s %s по цене %s 💵\n%s ⏱"

	textSide := "Куплен ➕"
	if orderInfo.Side == domain.OrderSideSell {
		textSide = "Продан ➖"
	}

	t, _ := time.Parse(time.RFC3339, orderInfo.Timestamp)
	loc, _ := time.LoadLocation("Europe/Moscow")
	t = t.In(loc)

	text := fmt.Sprintf(template, textSide, strings.ToUpper(orderInfo.Symbol[3:6]), strconv.FormatFloat(orderInfo.Price, 'f', -1, 64), t.Format(time.RFC1123))

	msg := tgbotapi.NewMessage(chatID, text)
	telegramBot.bot.Send(msg)
}
