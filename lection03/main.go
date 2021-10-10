package main

import (
	"context"
	"fmt"
	"lection03/domain"
	"lection03/generator"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

var tickers = []string{"AAPL", "SBER", "NVDA", "TSLA"}

func main() {
	logger := log.New()
	ctx, cancel := context.WithCancel(context.Background())

	pg := generator.NewPricesGenerator(generator.Config{
		Factor:  10,
		Delay:   time.Millisecond * 500,
		Tickers: tickers,
	})

	var wg sync.WaitGroup

	logger.Info("start prices generator...")
	prices := pg.Prices(ctx)
	wg.Add(4)
	candles1m := candles1mFromPrice(ctx, prices, &wg)
	candles2m := candlesFromCandles(ctx, domain.CandlePeriod2m, candles1m, &wg)
	candles10m := candlesFromCandles(ctx, domain.CandlePeriod10m, candles2m, &wg)
	freeChannel(ctx, candles10m, &wg)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done
	cancel()
	wg.Wait()
}

// Функция нужна чтобы очищать канал десяти-периодных свечей иначе получим дедлок
func freeChannel(ctx context.Context, candles <-chan domain.Candle, wg *sync.WaitGroup) {
	go func(ctx context.Context) {
		defer wg.Done()
		for {
			_, ok := <-candles
			if !ok {
				return
			}
		}
	}(ctx)
}

func writeToCsv(candle domain.Candle) {
	f, err := os.OpenFile(fmt.Sprintf("candles_%s.csv", candle.Period), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	if _, err = f.WriteString(fmt.Sprintf("%s,%s,%f,%f,%f,%f\n", candle.Ticker, candle.TS.Format(time.RFC3339), candle.Open, candle.High, candle.Low, candle.Close)); err != nil {
		panic(err)
	}
}

func newCandleByPrice(price domain.Price, period domain.CandlePeriod) domain.Candle {
	return domain.Candle{
		Ticker: price.Ticker,
		Period: period,
		Open:   price.Value,
		High:   price.Value,
		Low:    price.Value,
		Close:  price.Value,
		TS:     price.TS,
	}
}

func candles1mFromPrice(ctx context.Context, prices <-chan domain.Price, wg *sync.WaitGroup) <-chan domain.Candle {
	var candles = make(map[string]domain.Candle)

	output := make(chan domain.Candle)
	go func(ctx context.Context) {
		defer close(output)
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case price, ok := <-prices:
				if !ok {
					continue
				}
				if candle, ok := candles[price.Ticker]; ok {
					if isEqualPeriod(candle.TS, price.TS, domain.CandlePeriod1m) {
						if price.Value > candles[price.Ticker].High {
							candle.High = price.Value
						}
						if price.Value < candles[price.Ticker].Low {
							candle.Low = price.Value
						}
						candle.Close = price.Value
						candles[price.Ticker] = candle
						continue
					}
					writeToCsv(candle)
					output <- candle
				}
				candles[price.Ticker] = newCandleByPrice(price, domain.CandlePeriod1m)
			}
		}
	}(ctx)

	return output
}

func isEqualPeriod(firstTS time.Time, secondTS time.Time, period domain.CandlePeriod) bool {
	firstPeriod, _ := domain.PeriodTS(period, firstTS)
	secondPeriod, _ := domain.PeriodTS(period, secondTS)
	return firstPeriod == secondPeriod
}

func candlesFromCandles(ctx context.Context, period domain.CandlePeriod, candlesNth <-chan domain.Candle, wg *sync.WaitGroup) <-chan domain.Candle {
	var candles = make(map[string]domain.Candle)

	output := make(chan domain.Candle)
	go func(ctx context.Context) {
		defer close(output)
		defer wg.Done()
		for {
			newCandle, ok := <-candlesNth
			if !ok {
				for _, candle := range candles {
					writeToCsv(candle)
					output <- candle
				}
				return
			}
			if candle, ok := candles[newCandle.Ticker]; ok {
				if isEqualPeriod(candle.TS, newCandle.TS, period) {
					if candle.High < newCandle.High {
						candle.High = newCandle.High
					}
					if candle.Low > newCandle.Low {
						candle.Low = newCandle.Low
					}
					candle.Close = newCandle.Close
					candles[candle.Ticker] = candle
					continue
				}
				writeToCsv(candle)
				output <- candle
			}
			newCandle.Period = period
			candles[newCandle.Ticker] = newCandle
		}
	}(ctx)

	return output
}
