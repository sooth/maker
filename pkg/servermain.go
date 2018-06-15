// Copyright (C) 2018 Cranky Kernel
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package pkg

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"github.com/crankykernel/cryptotrader/binance"
	_ "github.com/mattn/go-sqlite3"
	"github.com/crankykernel/maker/pkg/log"
	"github.com/crankykernel/maker/pkg/config"
	"runtime"
	"os/exec"
	"sync"
	"path/filepath"
	"os"
	"time"
)

var ServerFlags struct {
	Host           string
	Port           int16
	ConfigFilename string
	LogFilename    string
	NoLog          bool
	OpenBrowser    bool
}

func getBinanceRestClient() *binance.RestClient {
	restClient := binance.NewAuthenticatedClient(
		config.GetString("binance.api.key"),
		config.GetString("binance.api.secret"))
	return restClient
}

type ApplicationContext struct {
	TradeService          *TradeService
	BinanceStreamManager  *BinanceStreamManager
	BinanceUserDataStream *BinanceUserDataStream
	OpenBrowser           bool
}

func restoreTrades(tradeService *TradeService) {
	binanceRestClient := getBinanceRestClient()
	tradeStates, err := DbRestoreTradeState()
	if err != nil {
		log.Fatalf("error: failed to restore trade state: %v", err)
	}

	tradeHistoryCache := map[string][]binance.TradeResponse{}

	for _, state := range (tradeStates) {
		trade := NewTradeWithState(state)
		tradeService.RestoreTrade(trade)

		if trade.State.Status == TradeStatusPendingBuy {
			order, err := binanceRestClient.GetOrderByOrderId(
				trade.State.Symbol, trade.State.BuyOrderId)
			if err != nil {
				log.WithError(err).Error("Failed to get order by ID.")
			}
			switch order.Status {
			case binance.OrderStatusNew:
				// No change.
			default:
				log.WithFields(log.Fields{
					"tradeId":     trade.State.LocalID,
					"orderStatus": order.Status,
					"symbol":      trade.State.Symbol,
					"tradeStatus": trade.State.Status,
				}).Warnf("Don't know how to restore pending buy trade.")
			}
		}

		if trade.State.Status == TradeStatusPendingSell {
			order, err := binanceRestClient.GetOrderByOrderId(
				trade.State.Symbol, trade.State.SellOrderId)
			if err != nil {
				log.WithError(err).Errorf(
					"Failed to find existing order %d for %s.",
					trade.State.SellOrderId, trade.State.Symbol)
			} else {
				if order.Status == binance.OrderStatusNew {
					// Unchanged.
				} else if order.Status == binance.OrderStatusCanceled {
					log.WithFields(log.Fields{
						"symbol":  state.Symbol,
						"tradeId": state.LocalID,
					}).Infof("Outstanding sell order has been canceled.")
					trade.State.Status = TradeStatusWatching
				} else if order.Status == binance.OrderStatusFilled {
					trades := tradeHistoryCache[state.Symbol]
					if trades == nil {
						trades, err = binanceRestClient.GetMytrades(state.Symbol, 0, -1)
						if err != nil {
							log.Errorf("Failed to get trades: %v", err)
						}
						tradeHistoryCache[state.Symbol] = trades
					}
					for _, _trade := range trades {
						if _trade.OrderID == state.SellOrderId {
							fill := OrderFill{
								Price:            _trade.Price,
								Quantity:         _trade.Quantity,
								CommissionAmount: _trade.Commission,
								CommissionAsset:  _trade.CommissionAsset,
							}
							trade.DoAddSellFill(fill)
						}
					}
					if trade.State.SellFillQuantity != trade.State.BuyFillQuantity {
						log.WithFields(log.Fields{
							"buyQuantity":  trade.State.BuyFillQuantity,
							"sellQuantity": trade.State.SellFillQuantity,
						}).Warnf("Order is filled but sell quantity != buy quantity.")
					} else {
						closeTime := time.Unix(0, order.TimeMillis*int64(time.Millisecond))
						log.WithFields(log.Fields{
							"symbol":    trade.State.Symbol,
							"closeTime": closeTime,
							"tradeId":   trade.State.LocalID,
						}).Infof("Closing trade.")
						tradeService.CloseTrade(trade, TradeStatusDone, closeTime)
					}
				} else {
					log.WithFields(log.Fields{
						"symbol":      state.Symbol,
						"tradeId":     state.LocalID,
						"orderStatus": order.Status,
					}).Warnf("Don't know how to restore trade in status %v: %s",
						order.Status, log.ToJson(order))
				}
			}
		}
	}
	log.Printf("Restored %d trade states.", len(tradeService.TradesByClientID))
}

func ServerMain() {

	log.SetLevel(log.LogLevelDebug)

	if !ServerFlags.NoLog {
		log.AddHook(log.NewFileOutputHook(ServerFlags.LogFilename))
	}

	if ServerFlags.Host != "127.0.0.1" {
		log.Fatal("Hosts other than 127.0.0.1 not allowed yet.")
	}

	applicationContext := &ApplicationContext{}
	applicationContext.BinanceStreamManager = NewBinanceStreamManager()

	DbOpen()

	tradeService := NewTradeService(applicationContext)
	applicationContext.TradeService = tradeService

	restoreTrades(tradeService)

	applicationContext.BinanceUserDataStream = NewBinanceUserDataStream()
	userStreamChannel := applicationContext.BinanceUserDataStream.Subscribe()
	go applicationContext.BinanceUserDataStream.Run()

	go func() {
		for {
			select {
			case event := <-userStreamChannel:
				switch event.EventType {
				case EventTypeExecutionReport:
					if err := DbSaveBinanceRawExecutionReport(event); err != nil {
						log.Println(err)
					}
					tradeService.OnExecutionReport(event)
				}
			}
		}
	}()

	router := mux.NewRouter()

	router.HandleFunc("/api/config", configHandler).Methods("GET")

	router.HandleFunc("/api/binance/buy", postBuyHandler(tradeService)).Methods("POST")
	router.HandleFunc("/api/binance/buy", deleteBuyHandler(tradeService)).Methods("DELETE")
	router.HandleFunc("/api/binance/sell", deleteSellHandler(tradeService)).Methods("DELETE")

	router.HandleFunc("/api/binance/trade/{tradeId}/stopLoss",
		updateTradeStopLossSettingsHandler(tradeService)).Methods("POST")
	router.HandleFunc("/api/binance/trade/{tradeId}/trailingStop",
		updateTradeTrailingStopSettingsHandler(tradeService)).Methods("POST")
	router.HandleFunc("/api/binance/trade/{tradeId}/limitSell",
		limitSellHandler(tradeService)).Methods("POST")
	router.HandleFunc("/api/binance/trade/{tradeId}/marketSell",
		marketSellHandler(tradeService)).Methods("POST")
	router.HandleFunc("/api/binance/trade/{tradeId}/archive",
		archiveTradeHandler(tradeService)).Methods("POST")
	router.HandleFunc("/api/binance/trade/{tradeId}/abandon",
		abandonTradeHandler(tradeService)).Methods("POST")

	router.HandleFunc("/api/binance/account/test",
		binanceTestHandler).Methods("GET")
	router.HandleFunc("/api/binance/config",
		saveBinanceConfigHandler).Methods("POST")
	router.HandleFunc("/api/config/preferences",
		savePreferencesHandler).Methods("POST");
	binanceApiProxyHandler := http.StripPrefix("/proxy/binance",
		binance.NewBinanceApiProxyHandler())
	router.PathPrefix("/proxy/binance").Handler(binanceApiProxyHandler)

	router.PathPrefix("/ws").Handler(NewUserWebSocketHandler(applicationContext))

	router.PathPrefix("/").HandlerFunc(staticAssetHandler())

	listenHostPort := fmt.Sprintf("%s:%d", ServerFlags.Host, ServerFlags.Port)
	log.Printf("Starting server on %s.", listenHostPort)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err := http.ListenAndServe(listenHostPort, router)
		if err != nil {
			log.Fatal("Failed to start server: ", err)
		}
	}()

	if ServerFlags.OpenBrowser {
		url := fmt.Sprintf("http://%s:%d", ServerFlags.Host, ServerFlags.Port)
		log.Info("Attempting to start browser.")
		go func() {
			if runtime.GOOS == "linux" {
				c := exec.Command("xdg-open", url)
				c.Run()
			} else if runtime.GOOS == "darwin" {
				c := exec.Command("open", url)
				c.Run()
			} else if runtime.GOOS == "windows" {
				cmd := "url.dll,FileProtocolHandler"
				runDll32 := filepath.Join(os.Getenv("SYSTEMROOT"), "System32",
					"rundll32.exe")
				c := exec.Command(runDll32, cmd, url)
				if err := c.Run(); err != nil {
					log.WithError(err).WithFields(log.Fields{
						"os": "windows",
					}).Errorf("Failed to start browser.")
				}
			}
		}()
	}

	wg.Wait()
}
