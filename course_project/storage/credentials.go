package storage

import "os"

type credentialsLogger interface {
	Panicf(format string, args ...interface{})
}

type Credentials struct {
	krakenPublicKey     string
	krakenSecretKey     string
	telegramBotAPIToken string
	databaseDSN         string
	websocketURL        string
	httpUrl             string
	logger              credentialsLogger
}

func NewCredentialsStorage(credentialsLogger credentialsLogger) *Credentials {
	credentials := Credentials{}

	credentials.krakenPublicKey = credentials.getKeyFromEnv("KRAKEN_API_PUBLIC_KEY")
	credentials.krakenSecretKey = credentials.getKeyFromEnv("KRAKEN_API_SECRET_KEY")
	credentials.telegramBotAPIToken = credentials.getKeyFromEnv("TELEGRAM_BOT_API_TOKEN")
	credentials.databaseDSN = credentials.getKeyFromEnv("DATABASE_DSN")
	credentials.websocketURL = "wss://demo-futures.kraken.com/ws/v1"
	credentials.httpUrl = "https://demo-futures.kraken.com/derivatives"
	credentials.logger = credentialsLogger

	return &credentials
}

func (credentials *Credentials) GetKrakenPublicKey() string {
	return credentials.krakenPublicKey
}

func (credentials *Credentials) GetKrakenSecretKey() string {
	return credentials.krakenSecretKey
}

func (credentials *Credentials) GetTelegramBotAPIToken() string {
	return credentials.telegramBotAPIToken
}

func (credentials *Credentials) GetDatabaseDSN() string {
	return credentials.databaseDSN
}

func (credentials *Credentials) GetWebsocketURL() string {
	return credentials.websocketURL
}

func (credentials *Credentials) GetHTTPUrl() string {
	return credentials.httpUrl
}

func (credentials *Credentials) getKeyFromEnv(keyName string) string {
	key := os.Getenv(keyName)
	if key == "" {
		credentials.logger.Panicf("Please set %s in system environment variables", keyName)
	}
	return key
}
