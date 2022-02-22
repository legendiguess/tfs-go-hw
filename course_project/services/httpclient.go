package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/legendiguess/kraken-trade-bot/domain"
)

type httpCredentials interface {
	GetKrakenPublicKey() string
	GetKrakenSecretKey() string
	GetHTTPUrl() string
}

type HTTPClient struct {
	httpCredentials httpCredentials
}

func NewHTTPClient(httpCredentials httpCredentials) *HTTPClient {
	return &HTTPClient{httpCredentials: httpCredentials}
}

func (httpClient *HTTPClient) GenerateAuthent(postData string, endpointPath string) string {
	concatenate := postData + endpointPath
	hashSHA256 := sha256.New()
	hashSHA256.Write([]byte(concatenate))

	decodedAPISecret, _ := base64.StdEncoding.DecodeString(httpClient.httpCredentials.GetKrakenSecretKey())

	h := hmac.New(sha512.New, decodedAPISecret)
	h.Write(hashSHA256.Sum(nil))

	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (httpClient *HTTPClient) sendRequest(method string, postData string, endPoint string, body io.Reader) (map[string]interface{}, error) {
	newRequest, _ := http.NewRequest(method, httpClient.httpCredentials.GetHTTPUrl()+endPoint+"?"+postData, body)

	newRequest.Header.Add("Authent", httpClient.GenerateAuthent(postData, endPoint))
	newRequest.Header.Add("APIKey", httpClient.httpCredentials.GetKrakenPublicKey())

	resp, err := http.DefaultClient.Do(newRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bytesAnswer, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var answerMap map[string]interface{}
	json.Unmarshal(bytesAnswer, &answerMap)
	return answerMap, nil
}

func (httpClient HTTPClient) Order(ticker string, side domain.OrderSide) (*domain.OrderInfo, error) {
	answer, err := httpClient.sendRequest("POST", fmt.Sprintf("orderType=mkt&symbol=%s&side=%s&size=1", ticker, side), "/api/v3/sendorder", nil)
	if err != nil {
		return nil, err
	}

	if value, ok := answer["result"]; ok {
		if value == "success" {
			sendStatus := answer["sendStatus"].(map[string]interface{})
			orderEvents := sendStatus["orderEvents"].([]interface{})
			orderEvent := orderEvents[0].(map[string]interface{})
			if orderEvent["type"] == "EXECUTION" {
				orderPriorExecution := orderEvent["orderPriorExecution"].(map[string]interface{})

				var orderInfo = domain.OrderInfo{
					OrderID:     orderPriorExecution["orderId"].(string),
					ExecutionID: orderEvent["executionId"].(string),
					Price:       orderEvent["price"].(float64),
					Amount:      uint64(orderEvent["amount"].(float64)),
					Type:        orderPriorExecution["type"].(string),
					Symbol:      orderPriorExecution["symbol"].(string),
					Side:        domain.OrderSide(orderPriorExecution["side"].(string)),
					Quantity:    uint64(orderPriorExecution["quantity"].(float64)),
					LimitPrice:  orderPriorExecution["limitPrice"].(float64),
					Timestamp:   orderPriorExecution["timestamp"].(string),
				}

				return &orderInfo, nil
			}
			return nil, errors.New(orderEvent["reason"].(string))
		}
	}

	return nil, errors.New("Something wrong with request parameters")
}
