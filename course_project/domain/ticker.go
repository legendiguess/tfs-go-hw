package domain

type Ticker map[string]interface{}

func (ticker *Ticker) GetAsk() float64 {
	return (*ticker)["ask"].(float64)
}

func (ticker *Ticker) GetBid() float64 {
	return (*ticker)["bid"].(float64)
}

func (ticker *Ticker) GetSymbol() string {
	return (*ticker)["product_id"].(string)
}
