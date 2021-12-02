package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/legendiguess/kraken-trade-bot/domain"
	"nhooyr.io/websocket"
)

type websocketCredentials interface {
	GetWebsocketURL() string
}

type websocketClientLogger interface {
	Panicf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Printf(format string, args ...interface{})
}

type WebsocketClient struct {
	connection *websocket.Conn
	context    context.Context
	logger     websocketClientLogger
}

// Create connected websocket client
func NewWebsocketClient(ctx context.Context, websocketCredentials websocketCredentials, websocketClientLogger websocketClientLogger) *WebsocketClient {
	var websocketClient = WebsocketClient{logger: websocketClientLogger}
	websocketClient.context = ctx

	var err error

	for {
		websocketClient.connection, _, err = websocket.Dial(websocketClient.context, websocketCredentials.GetWebsocketURL(), nil)
		if err != nil {
			time.Sleep(1 * time.Second)
			websocketClient.logger.Debugf("Attempting to establish a websocket connection...")
			continue
		}
		break
	}
	websocketClient.logger.Debugf("Websocket connection established")

	// Ping every 30 sec
	go func() {
		for {
			select {
			case <-websocketClient.context.Done():
				return
			default:
				time.Sleep(time.Second * 30)
				websocketClient.connection.Ping(websocketClient.context)
			}
		}
	}()

	return &websocketClient
}

func (websocketClient *WebsocketClient) UnsubscribeFromTicker(productIDs []string) {
	bytes, err := json.Marshal(map[string]interface{}{
		"event":       "unsubscribe",
		"feed":        "ticker",
		"product_ids": productIDs,
	})

	if err != nil {
		websocketClient.logger.Panicf("%v", err)
	}

	websocketClient.connection.Write(websocketClient.context, websocket.MessageText, bytes)

	websocketClient.logger.Printf("Unsubscribed from %s ticker", productIDs[0])
}

func (websocketClient *WebsocketClient) SubscribeToTicker(productIDs []string) {
	bytes, err := json.Marshal(map[string]interface{}{
		"event":       "subscribe",
		"feed":        "ticker",
		"product_ids": productIDs,
	})

	if err != nil {
		websocketClient.logger.Panicf("%v", err)
	}

	websocketClient.connection.Write(websocketClient.context, websocket.MessageText, bytes)

	websocketClient.logger.Printf("Subscribed to %s ticker", productIDs[0])
}

func (websocketClient WebsocketClient) GetTickerChannel() <-chan domain.Ticker {
	tickers := make(chan domain.Ticker)

	go func() {
		defer close(tickers)

		for {
			select {
			case <-websocketClient.context.Done():
				return
			default:
				_, bytes, err := websocketClient.connection.Read(websocketClient.context)

				if err != nil {
					return
				}

				var newTicker domain.Ticker
				err = json.Unmarshal(bytes, &newTicker)

				if err != nil {
					continue
				}

				// Checking if it is real ticker
				if _, ok := newTicker["product_id"]; ok {
					tickers <- newTicker
				}
			}
		}
	}()

	return tickers
}

func (websocketClient *WebsocketClient) CloseConnection() {
	websocketClient.connection.Close(websocket.StatusNormalClosure, "")
}
