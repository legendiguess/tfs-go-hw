package domain

type OrderSide string

const (
	OrderSideBuy  = OrderSide("buy")
	OrderSideSell = OrderSide("sell")
)

type OrderInfo struct {
	OrderID     string `json:"order_id"`
	ExecutionID string `json:"executionId"`
	Price       float64
	Amount      uint64
	Type        string
	Symbol      string
	Side        OrderSide
	Quantity    uint64
	LimitPrice  float64
	Timestamp   string
}
