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
	wg.Add(7)
	candles1m := candles1mFromPrice(prices, &wg)
	candles1m = writeCandlesToCSV(candles1m, &wg)
	candles2m := candlesFromCandles(domain.CandlePeriod2m, candles1m, &wg)
	candles2m = writeCandlesToCSV(candles2m, &wg)
	candles10m := candlesFromCandles(domain.CandlePeriod10m, candles2m, &wg)
	candles10m = writeCandlesToCSV(candles10m, &wg)
	freeChannel(candles10m, &wg)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done
	cancel()
	wg.Wait()
}

// Функция нужна чтобы очищать канал десяти-периодных свечей иначе получим дедлок
func freeChannel(candles <-chan domain.Candle, wg *sync.WaitGroup) {
	go func() {
		defer wg.Done()
		for range candles {
		}
	}()
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

func writeCandlesToCSV(candles <-chan domain.Candle, wg *sync.WaitGroup) <-chan domain.Candle {
	output := make(chan domain.Candle)
	go func() {
		defer close(output)
		defer wg.Done()
		for candle := range candles {
			writeToCsv(candle)
			output <- candle
		}
	}()

	return output
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

func candles1mFromPrice(prices <-chan domain.Price, wg *sync.WaitGroup) <-chan domain.Candle {
	var candles = make(map[string]domain.Candle)

	output := make(chan domain.Candle)
	go func() {
		defer close(output)
		defer wg.Done()
		for price := range prices {
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
				output <- candle
			}
			candles[price.Ticker] = newCandleByPrice(price, domain.CandlePeriod1m)
		}
	}()

	return output
}

func isEqualPeriod(firstTS time.Time, secondTS time.Time, period domain.CandlePeriod) bool {
	firstPeriod, _ := domain.PeriodTS(period, firstTS)
	secondPeriod, _ := domain.PeriodTS(period, secondTS)
	return firstPeriod == secondPeriod
}

func candlesFromCandles(period domain.CandlePeriod, candlesNth <-chan domain.Candle, wg *sync.WaitGroup) <-chan domain.Candle {
	var candles = make(map[string]domain.Candle)

	output := make(chan domain.Candle)
	go func() {
		defer close(output)
		defer wg.Done()
		for newCandle := range candlesNth {
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
				output <- candle
			}
			newCandle.Period = period
			candles[newCandle.Ticker] = newCandle
		}
		for _, candle := range candles {
			output <- candle
		}
	}()

	return output
}
