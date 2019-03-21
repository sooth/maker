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

package server

import (
	"encoding/json"
	"fmt"
	"github.com/crankykernel/binanceapi-go"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mholt/certmagic"
	"gitlab.com/crankykernel/maker/go/binanceex"
	"gitlab.com/crankykernel/maker/go/clientnotificationservice"
	"gitlab.com/crankykernel/maker/go/context"
	"gitlab.com/crankykernel/maker/go/db"
	"gitlab.com/crankykernel/maker/go/gencert"
	"gitlab.com/crankykernel/maker/go/healthservice"
	"gitlab.com/crankykernel/maker/go/log"
	"gitlab.com/crankykernel/maker/go/tradeservice"
	"gitlab.com/crankykernel/maker/go/version"
	stdlog "log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"time"
)

var ServerFlags struct {
	Host           string
	Port           int16
	ConfigFilename string
	NoLog          bool
	OpenBrowser    bool
	DataDirectory  string
	TLS            bool
	NoTLS          bool
	LetsEncrypt    bool
	LeHostname     string
	ItsAllMyFault  bool
	EnableAuth     bool
	NoAuth         bool
}

func initBinanceExchangeInfoService() *binanceex.ExchangeInfoService {
	exchangeInfoService := binanceex.NewExchangeInfoService()
	if err := exchangeInfoService.Update(); err != nil {
		log.WithError(err).Errorf("Binance exchange info server failed to update")
	}
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			if err := exchangeInfoService.Update(); err != nil {
				log.WithError(err).Errorf("Binance exchange info server failed to update")
			}
		}
	}()
	return exchangeInfoService
}

type LogInterceptor struct{}

func (l *LogInterceptor) Write(p []byte) (n int, err error) {
	log.Printf("%s", string(p))
	return 0, nil
}

func ServerMain() {

	stdlog.SetOutput(&LogInterceptor{})
	stdlog.SetFlags(0)

	log.SetLevel(log.LogLevelDebug)

	if _, err := os.Stat(ServerFlags.DataDirectory); err != nil {
		if err := os.Mkdir(ServerFlags.DataDirectory, 0700); err != nil {
			log.Fatalf("Failed to create data directory %s: %v", ServerFlags.DataDirectory, err)
		}
	}

	if !ServerFlags.NoLog {
		log.AddHook(log.NewFileOutputHook(path.Join(ServerFlags.DataDirectory, "maker.log")))
	}

	log.Infof("This is Maker version %s (git revision %s)",
		version.Version, version.GitRevision)

	ServerFlags.ConfigFilename = path.Join(ServerFlags.DataDirectory, "maker.yaml")

	if ServerFlags.Host != "127.0.0.1" {
		if !ServerFlags.EnableAuth && !ServerFlags.NoAuth {
			log.Fatalf("Authentication must be enabled to listen on anything other than 127.0.0.1")
		}
		if !ServerFlags.LetsEncrypt && (!ServerFlags.TLS && !ServerFlags.NoTLS) {
			log.Fatalf("TLS must be enabled to list on anything other than 127.0.0.1")
		}
		if !ServerFlags.ItsAllMyFault {
			log.Fatalf("Secret command line argument for non 127.0.0.1 listen not set. See documentation.")
		}
	}

	if ServerFlags.LetsEncrypt {
		if ServerFlags.LeHostname == "" {
			log.Fatalf("Lets Encrypt support requires --letsencrypt-hostname")
		}
		if !ServerFlags.EnableAuth {
			log.Fatalf("Authentication required for Lets Encrypt support")
		}
		if !ServerFlags.ItsAllMyFault {
			log.Fatalf("Secret command line argument Lets Encrypt support not set. See documentation.")
		}
		log.Warnf("Lets Encrypt automatically listens on port 443, --port is ignored.")
		log.Warnf("Lets Encrypt support enables remote acccess, --host is ignored.")
	} else if ServerFlags.TLS {
		pemFilename := fmt.Sprintf("%s/maker.pem", ServerFlags.DataDirectory)
		if _, err := os.Stat(pemFilename); err != nil {
			gencert.GenCertMain(gencert.Flags{
				Host:     &gencert.DEFAULT_HOST,
				Org:      &gencert.DEFAULT_ORG,
				Filename: &pemFilename,
			}, []string{})
		}
	}

	applicationContext := &context.ApplicationContext{}
	applicationContext.BinanceTradeStreamManager = binanceex.NewTradeStreamManager()

	db.DbOpen(ServerFlags.DataDirectory)

	tradeService := tradeservice.NewTradeService(applicationContext.BinanceTradeStreamManager)
	applicationContext.TradeService = tradeService

	restoreTrades(tradeService)

	binanceExchangeInfoService := initBinanceExchangeInfoService()
	binancePriceService := binanceex.NewBinancePriceService(binanceExchangeInfoService)

	clientNotificationService := clientnotificationservice.New()
	healthService := healthservice.New()

	applicationContext.BinanceUserDataStream = binanceex.NewBinanceUserDataStream(
		clientNotificationService, healthService)
	userStreamChannel := applicationContext.BinanceUserDataStream.Subscribe("main")
	go applicationContext.BinanceUserDataStream.Run()

	go func() {
		for {
			client := binanceapi.NewRestClient()
			requestStart := time.Now()
			response, err := client.GetTime()
			if err != nil {
				log.WithError(err).Errorf("Failed to get from Binance API")
				time.Sleep(1 * time.Minute)
				continue
			}

			roundTripTime := time.Now().Sub(requestStart)
			now := time.Now().UnixNano() / int64(time.Millisecond)
			diff := math.Abs(float64(now - response.ServerTime))
			logFields := log.Fields{
				"roundTripTime":          fmt.Sprintf("%v", roundTripTime),
				"binanceTimeDifferentMs": fmt.Sprintf("%v", diff),
			}
			if diff > 999 {
				log.WithFields(logFields).Warnf("Time difference from Binance servers may be too large; order may fail")
				clientNotificationService.Broadcast(clientnotificationservice.NewNotice(clientnotificationservice.LevelWarning,
					"Time difference between Binance and Maker server too large, orders may fail."))
			} else {
				log.WithFields(logFields).Infof("Binance time check")
			}
			time.Sleep(1 * time.Minute)
		}
	}()

	go func() {
		for {
			select {
			case event := <-userStreamChannel:
				switch event.EventType {
				case binanceex.EventTypeExecutionReport:
					if err := db.DbSaveBinanceRawExecutionReport(event.EventTime, event.Raw); err != nil {
						log.Println(err)
					}
					tradeService.OnExecutionReport(event)
				}
			}
		}
	}()

	router := mux.NewRouter()

	var authenticator *Authenticator = nil
	if ServerFlags.EnableAuth {
		authenticator = NewAuthenticator(ServerFlags.ConfigFilename)
		router.Use(authenticator.Middleware)
	}

	router.HandleFunc("/api/config", configHandler).Methods("GET")
	router.HandleFunc("/api/version", VersionHandler).Methods("GET")
	router.HandleFunc("/api/time", TimeHandler).Methods("GET")
	router.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		type LoginForm struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		var loginForm LoginForm
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&loginForm); err != nil {
			log.WithError(err).Errorf("Failed to decode login form")
			WriteJsonError(w, http.StatusInternalServerError, "error decoding login form")
			return
		}
		if authenticator == nil {
			WriteJsonResponse(w, http.StatusOK, map[string]interface{}{})
			return
		}
		sessionId, err := authenticator.Login(loginForm.Username, loginForm.Password)
		if err != nil {
			log.WithError(err).WithField("username", loginForm.Username).
				Errorf("Login failed")
			WriteJsonError(w, http.StatusUnauthorized, "authentication failed")
			return
		}
		WriteJsonResponse(w, http.StatusOK, map[string]interface{}{
			"sessionId": sessionId,
		})
	})

	router.HandleFunc("/api/binance/buy", PostBuyHandler(tradeService, binancePriceService)).Methods("POST")
	router.HandleFunc("/api/binance/buy", deleteBuyHandler(tradeService)).Methods("DELETE")
	router.HandleFunc("/api/binance/sell", DeleteSellHandler(tradeService)).Methods("DELETE")

	// Set/change stop-loss on a trade.
	router.HandleFunc("/api/binance/trade/{tradeId}/stopLoss",
		updateTradeStopLossSettingsHandler(tradeService)).Methods("POST")

	router.HandleFunc("/api/binance/trade/{tradeId}/trailingProfit",
		updateTradeTrailingProfitSettingsHandler(tradeService)).Methods("POST")

	// Limit sell at percent.
	router.HandleFunc("/api/binance/trade/{tradeId}/limitSellByPercent",
		limitSellByPercentHandler(tradeService)).Methods("POST")

	// Limit sell at price.
	router.HandleFunc("/api/binance/trade/{tradeId}/limitSellByPrice",
		limitSellByPriceHandler(tradeService)).Methods("POST")

	router.HandleFunc("/api/binance/trade/{tradeId}/marketSell",
		marketSellHandler(tradeService)).Methods("POST")
	router.HandleFunc("/api/binance/trade/{tradeId}/archive",
		archiveTradeHandler(tradeService)).Methods("POST")
	router.HandleFunc("/api/binance/trade/{tradeId}/abandon",
		abandonTradeHandler(tradeService)).Methods("POST")

	// Handlers that proxy requests to Binance.
	binanceProxyHandlers := NewBinanceProxyHandlers()
	binanceProxyHandlers.RegisterHandlers(router)

	router.HandleFunc("/api/trade/query", queryTradesHandler).
		Methods("GET")
	router.HandleFunc("/api/trade/{tradeId}",
		getTradeHandler).Methods("GET")

	router.HandleFunc("/api/binance/account/test",
		BinanceTestHandler).Methods("GET")
	router.HandleFunc("/api/binance/config",
		SaveBinanceConfigHandler).Methods("POST")
	router.HandleFunc("/api/config/preferences",
		SavePreferencesHandler).Methods("POST")

	binanceApiProxyHandler := http.StripPrefix("/proxy/binance",
		binanceapi.NewBinanceApiProxyHandler())
	router.PathPrefix("/proxy/binance").Handler(binanceApiProxyHandler)

	router.PathPrefix("/ws").Handler(NewUserWebSocketHandler(applicationContext,
		clientNotificationService, healthService))

	router.PathPrefix("/").HandlerFunc(staticAssetHandler())

	if ServerFlags.OpenBrowser {
		go func() {
			time.Sleep(time.Millisecond * 500)
			url := fmt.Sprintf("http://%s:%d", ServerFlags.Host, ServerFlags.Port)
			log.Info("Attempting to start browser.")
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

	var err error = nil
	if ServerFlags.LetsEncrypt {
		certmagic.Agreed = true
		if os.Getenv("LETSENCRYPT_STAGING") != "" {
			certmagic.CA = certmagic.LetsEncryptStagingCA
		}
		certmagic.DefaultStorage = &certmagic.FileStorage{
			Path: fmt.Sprintf("%s/certmagic", ServerFlags.DataDirectory),
		}

		err = certmagic.HTTPS([]string{ServerFlags.LeHostname}, router)
	} else if ServerFlags.TLS {
		listenHostPort := fmt.Sprintf("%s:%d", ServerFlags.Host, ServerFlags.Port)
		log.Printf("Starting HTTPS server on %s.", listenHostPort)
		pemFilename := fmt.Sprintf("%s/maker.pem", ServerFlags.DataDirectory)
		err = http.ListenAndServeTLS(listenHostPort, pemFilename, pemFilename, router)
	} else {
		listenHostPort := fmt.Sprintf("%s:%d", ServerFlags.Host, ServerFlags.Port)
		log.Printf("Starting HTTP server on %s.", listenHostPort)
		err = http.ListenAndServe(listenHostPort, router)
	}

	if err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
