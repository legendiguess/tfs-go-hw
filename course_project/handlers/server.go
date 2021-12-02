package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/legendiguess/kraken-trade-bot/domain"
)

type instrumentService interface {
	SaveInstrument(newInstrument domain.InstrumentConfig)
	GetInstrument() (domain.InstrumentConfig, bool)
}

type websocketClientService interface {
	SubscribeToTicker(productIDs []string)
	UnsubscribeFromTicker(productIDs []string)
}

type serverLogger interface {
	Panic(args ...interface{})
}

type Server struct {
	instrumentService instrumentService
	websocketClient   websocketClientService
	logger            serverLogger
}

func NewServer(instrumentService instrumentService, websocketClient websocketClientService, serverLogger serverLogger) *Server {
	server := Server{
		instrumentService: instrumentService,
		websocketClient:   websocketClient,
		logger:            serverLogger,
	}

	go func() {
		server.logger.Panic(http.ListenAndServe(":5000", server.Routes()))
	}()

	instrument, ok := instrumentService.GetInstrument()
	if ok {
		server.websocketClient.SubscribeToTicker([]string{instrument.Symbol})
	}

	return &server
}

func (server *Server) Routes() chi.Router {
	root := chi.NewRouter()

	root.Use(middleware.Logger)
	root.Put("/instrument", server.instrumentUpdate)

	root.Mount("/", root)

	return root
}

func (server *Server) instrumentUpdate(w http.ResponseWriter, r *http.Request) {
	d, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var instrumentConfig domain.InstrumentConfig
	err = json.Unmarshal(d, &instrumentConfig)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	oldInstrument, _ := server.instrumentService.GetInstrument()
	server.websocketClient.UnsubscribeFromTicker([]string{oldInstrument.Symbol})
	server.instrumentService.SaveInstrument(instrumentConfig)
	server.websocketClient.SubscribeToTicker([]string{instrumentConfig.Symbol})

	w.WriteHeader(http.StatusOK)
}
