package services

import (
	"github.com/legendiguess/kraken-trade-bot/domain"
)

type orderInfosStorage interface {
	NewOrderInfo(orderInfo *domain.OrderInfo)
}

type OrderInfosService struct {
	storage orderInfosStorage
}

func NewOrderInfosService(orderInfosStorage orderInfosStorage) *OrderInfosService {
	return &OrderInfosService{storage: orderInfosStorage}
}

func (orderInfosService *OrderInfosService) NewOrderInfo(orderInfo *domain.OrderInfo) {
	orderInfosService.storage.NewOrderInfo(orderInfo)
}
