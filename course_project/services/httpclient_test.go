package services_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/legendiguess/kraken-trade-bot/domain"
	"github.com/legendiguess/kraken-trade-bot/services"
)

type testHTTPCredentials struct {
	url string
}

func (httpCredentials *testHTTPCredentials) GetKrakenSecretKey() string {
	return "rttp4AzwRfYEdQ7R7X8Z/04Y4TZPa97pqCypi3xXxAqftygftnI6H9yGV+OcUOOJeFtZkr8mVwbAndU3Kz4Q+eG"
}

func (httpCredentials *testHTTPCredentials) GetKrakenPublicKey() string {
	return ""
}

func (httpCredentials *testHTTPCredentials) GetHTTPUrl() string {
	return httpCredentials.url
}

func TestGenerateAuthent(t *testing.T) {
	httpClient := services.NewHTTPClient(&testHTTPCredentials{})

	postData := "symbol=fi_xbtusd_180615"
	endpointPath := "/api/v3/orderbook"

	authent := httpClient.GenerateAuthent(postData, endpointPath)

	assert.Equal(t, "SnGkB1bzQClAvRtnum0VHEz76mFNIqUE+GkIBLKtsGx8cuyKwRxglzFXBEpAWFP70n3o0BS/i/5Q1uKSL4nYkQ==", authent)
}

func TestOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		answer := `{"result":"success","sendStatus":{"order_id":"888cbc74-4048-45c9-8e60-e520a6a474af","status":"placed","receivedTime":"2021-11-25T19:51:10.056Z","orderEvents":[{"executionId":"058f949f-e731-4500-b897-8e57966d630e","price":59238.5,"amount":1,"orderPriorEdit":null,"orderPriorExecution":{"orderId":"888cbc74-4048-45c9-8e60-e520a6a474af","cliOrdId":null,"type":"ioc","symbol":"pi_xbtusd","side":"sell","quantity":1,"filled":0,"limitPrice":30000,"reduceOnly":false,"timestamp":"2021-11-25T19:51:10.056Z","lastUpdateTimestamp":"2021-11-25T19:51:10.056Z"},"takerReducedQuantity":null,"type":"EXECUTION"}]},"serverTime":"2021-11-25T19:51:10.241Z"}`
		_, _ = resp.Write([]byte(answer))
	}))
	defer server.Close()

	httpClient := services.NewHTTPClient(&testHTTPCredentials{url: server.URL})
	_, err := httpClient.Order("pi_xbtusd", domain.OrderSideSell)

	assert.Nil(t, err)
}
