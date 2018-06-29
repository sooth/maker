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
	"net/http"
	"fmt"
	"io/ioutil"
	"encoding/json"
	"strconv"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
	"gitlab.com/crankykernel/maker/pkg/log"
	"gitlab.com/crankykernel/cryptotrader/binance"
	"github.com/gobuffalo/packr"
	"gitlab.com/crankykernel/maker/pkg/handlers"
	"gitlab.com/crankykernel/maker/pkg/maker"
	"time"
)

func archiveTradeHandler(tradeService *TradeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		tradeId := vars["tradeId"]
		if tradeId == "" {
			handlers.WriteJsonError(w, http.StatusBadRequest, "tradeId required")
			return
		}

		logFields := log.Fields{
			"tradeId": tradeId,
		}

		trade := tradeService.FindTradeByLocalID(tradeId)
		if trade == nil {
			log.WithFields(logFields).
				Warn("Failed to archive trade, tradeId not found.")
			handlers.WriteJsonError(w, http.StatusNotFound, "trade not found")
			return
		}

		if err := tradeService.ArchiveTrade(trade); err != nil {
			log.WithFields(logFields).
				WithError(err).Error("Failed to archive trade.")
			handlers.WriteJsonError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.WithFields(logFields).Info("Trade archived.")
	}
}

func abandonTradeHandler(tradeService *TradeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		tradeId := vars["tradeId"]
		if tradeId == "" {
			handlers.WriteJsonError(w, http.StatusBadRequest, "tradeId required")
			return
		}

		logFields := log.Fields{
			"tradeId": tradeId,
		}

		trade := tradeService.FindTradeByLocalID(tradeId)
		if trade == nil {
			log.WithFields(logFields).
				Warn("Failed to abandon trade, tradeId not found.")
			handlers.WriteJsonError(w, http.StatusNotFound, "trade not found")
			return
		}

		tradeService.AbandonTrade(trade)
		log.WithFields(logFields).Info("Trade abandoned.")
	}
}

func updateTradeStopLossSettingsHandler(tradeService *TradeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		if err = r.ParseForm(); err != nil {
			handlers.WriteBadRequestError(w)
			return
		}

		var tradeId string
		var enable bool
		var percent float64

		vars := mux.Vars(r)
		tradeId = vars["tradeId"]
		if tradeId == "" {
			handlers.WriteBadRequestError(w)
			return
		}

		if enable, err = strconv.ParseBool(r.FormValue("enable")); err != nil {
			handlers.WriteBadRequestError(w)
			return
		}
		if percent, err = strconv.ParseFloat(r.FormValue("percent"), 64); err != nil {
			handlers.WriteBadRequestError(w)
			return
		}

		trade := tradeService.FindTradeByLocalID(tradeId)
		if trade == nil {
			log.Printf("Failed to find trade with ID %s.", tradeId)
			handlers.WriteJsonError(w, http.StatusNotFound, "")
		}

		log.Printf("Updating stop loss for trade %s: enable=%v; percent=%v",
			tradeId, enable, percent)
		tradeService.UpdateStopLoss(trade, enable, percent)
	}
}

func updateTradeTrailingProfitSettingsHandler(tradeService *TradeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		if err = r.ParseForm(); err != nil {
			handlers.WriteBadRequestError(w)
			return
		}

		var tradeId string
		var enable bool
		var percent float64
		var deviation float64

		vars := mux.Vars(r)
		tradeId = vars["tradeId"]
		if tradeId == "" {
			handlers.WriteBadRequestError(w)
			return
		}

		if enable, err = strconv.ParseBool(r.FormValue("enable")); err != nil {
			handlers.WriteBadRequestError(w)
			return
		}
		if percent, err = strconv.ParseFloat(r.FormValue("percent"), 64); err != nil {
			handlers.WriteBadRequestError(w)
			return
		}
		if deviation, err = strconv.ParseFloat(r.FormValue("deviation"), 64); err != nil {
			handlers.WriteBadRequestError(w)
			return
		}

		trade := tradeService.FindTradeByLocalID(tradeId)
		if trade == nil {
			log.Printf("Failed to find trade with ID %s.", tradeId)
			handlers.WriteJsonError(w, http.StatusNotFound, "")
		}

		tradeService.UpdateTrailingProfit(trade, enable, percent, deviation)
	}
}

func deleteBuyHandler(tradeService *TradeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			handlers.WriteBadRequestError(w)
			return
		}

		tradeId := r.FormValue("trade_id")
		if tradeId == "" {
			handlers.WriteBadRequestError(w)
		}

		trade := tradeService.FindTradeByLocalID(tradeId)
		if trade == nil {
			log.WithFields(log.Fields{
				"tradeId": tradeId,
			}).Warnf("Failed to cancel buy order. Trade ID not found.")
			handlers.WriteBadRequestError(w)
			return
		}

		log.WithFields(log.Fields{
			"symbol":  trade.State.Symbol,
			"tradeId": tradeId,
		}).Infof("Cancelling buy order.")

		response, err := getBinanceRestClient().CancelOrder(trade.State.Symbol,
			trade.State.BuyOrderId)
		if err != nil {
			log.WithError(err).Errorf("Failed to cancel buy order.")
			return
		}

		handlers.WriteJsonResponse(w, http.StatusOK, response)
	}
}

func deleteSellHandler(tradeService *TradeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		tradeId := r.FormValue("trade_id")

		if tradeId == "" {
			handlers.WriteBadRequestError(w)
			return
		}

		trade := tradeService.FindTradeByLocalID(tradeId)
		if trade == nil {
			log.WithFields(log.Fields{
				"tradeId": tradeId,
			}).Warnf("Failed to cancel sell order. No trade found for ID.")
			handlers.WriteBadRequestError(w)
			return
		}

		log.WithFields(log.Fields{
			"symbol":      trade.State.Symbol,
			"tradeId":     trade.State.TradeID,
			"sellOrderId": trade.State.SellOrderId,
		}).Infof("Cancelling sell order.")

		response, err := getBinanceRestClient().CancelOrder(trade.State.Symbol,
			trade.State.SellOrderId)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"symbol":      trade.State.Symbol,
				"tradeId":     tradeId,
				"sellOrderId": trade.State.SellOrderId,
			}).Error("Failed to cancel sell order.")
			handlers.WriteJsonError(w, http.StatusBadRequest,
				fmt.Sprintf("Failed to cancel sell order: %s", string(err.Error())))
			return
		}

		handlers.WriteJsonResponse(w, http.StatusOK, response)
	}
}

func PostBuyHandler(tradeService *TradeService) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		params := binance.OrderParameters{
			Side:        binance.OrderSideBuy,
			Type:        binance.OrderTypeLimit,
			TimeInForce: binance.TimeInForceGTC,
		}

		var requestBody handlers.BuyOrderRequest
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&requestBody); err != nil {
			log.Printf("error: failed to decode request body: %v", err)
			handlers.WriteBadRequestError(w)
			return
		}

		// Validate price source.
		switch requestBody.PriceSource {
		case maker.PriceSourceLast:
		case maker.PriceSourceBestBid:
		case maker.PriceSourceBestAsk:
		case maker.PriceSourceManual:
		case "":
			handlers.WriteJsonError(w, http.StatusBadRequest, "missing required parameter: priceSource")
			return
		default:
			handlers.WriteJsonError(w, http.StatusBadRequest,
				fmt.Sprintf("invalid value for priceSource: %v", requestBody.PriceSource))
			return
		}

		params.Symbol = requestBody.Symbol
		params.Quantity = requestBody.Quantity

		orderId, err := tradeService.MakeOrderID()
		if err != nil {
			log.WithError(err).Errorf("Failed to create order ID.")
			handlers.WriteJsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		params.NewClientOrderId = orderId

		trade := maker.NewTrade()
		trade.State.Symbol = params.Symbol
		trade.AddClientOrderID(params.NewClientOrderId)

		buyService := NewBinanceBuyService()

		switch requestBody.PriceSource {
		case maker.PriceSourceManual:
			log.Infof("Using manual price of %v", requestBody.Price)
			params.Price = requestBody.Price
		default:
			params.Price, err = buyService.GetPrice(params.Symbol, requestBody.PriceSource)
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"priceSource": requestBody.PriceSource,
					"symbol":      params.Symbol,
				}).Error("Failed to get buy price.")
				handlers.WriteJsonError(w, http.StatusInternalServerError,
					fmt.Sprintf("Failed to get price: %v", err))
				return
			}
		}

		log.WithFields(log.Fields{
			"symbol":      params.Symbol,
			"priceSource": requestBody.PriceSource,
			"price":       params.Price,
		}).Debug("Got purchase price for symbol.")

		if requestBody.StopLossEnabled {
			trade.SetStopLoss(requestBody.StopLossEnabled,
				requestBody.StopLossPercent)
		}

		if requestBody.LimitSellEnabled {
			trade.SetLimitSell(requestBody.LimitSellEnabled,
				requestBody.LimitSellPercent)
		}

		if requestBody.TrailingProfitEnabled {
			trade.SetTrailingProfit(requestBody.TrailingProfitEnabled,
				requestBody.TrailingProfitPercent,
				requestBody.TrailingProfitDeviation)
		}

		tradeId := tradeService.AddNewTrade(trade)

		log.WithFields(log.Fields{
			"tradeId":       tradeId,
			"symbol":        params.Symbol,
			"price":         params.Price,
			"quantity":      params.Quantity,
			"type":          params.Type,
			"clientOrderId": params.NewClientOrderId,
		}).Infof("Posting BUY order for %s", params.Symbol)

		response, err := getBinanceRestClient().PostOrder(params)
		if err != nil {
			log.WithError(err).
				Errorf("Failed to post buy order.")
			switch err := err.(type) {
			case *binance.RestApiError:
				log.Debugf("Forwarding Binance error repsonse.")
				w.WriteHeader(response.StatusCode)
				w.Write(err.Body)
			default:
				handlers.WriteJsonResponse(w, http.StatusInternalServerError,
					err.Error())
			}
			if trade != nil {
				tradeService.FailTrade(trade)
			}
			return
		}

		data, err := ioutil.ReadAll(response.Body)
		var buyResponse binance.PostOrderResponse
		if err := json.Unmarshal(data, &buyResponse); err != nil {
			log.Printf("error: failed to decode buy order response: %v", err)
		}
		log.WithFields(log.Fields{
			"tradeId": tradeId,
		}).Debugf("Decoded BUY response: %s", log.ToJson(buyResponse))

		handlers.WriteJsonResponse(w, http.StatusOK, handlers.BuyOrderResponse{
			TradeID: tradeId,
		})
	}
}

func limitSellByPercentHandler(tradeService *TradeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		if err := r.ParseForm(); err != nil {
			handlers.WriteBadRequestError(w)
			return
		}

		tradeId := vars["tradeId"]
		if tradeId == "" {
			handlers.WriteBadRequestError(w)
			return
		}

		percent, err := strconv.ParseFloat(r.FormValue("percent"), 64)
		if err != nil {
			handlers.WriteBadRequestError(w)
			return
		}

		trade := tradeService.FindTradeByLocalID(tradeId)
		if trade == nil {
			handlers.WriteJsonError(w, http.StatusNotFound, "")
			return
		}

		startTime := time.Now()

		if trade.State.Status == maker.TradeStatusPendingSell {
			log.Printf("Cancelling existing sell order.");
			tradeService.CancelSell(trade)
		}

		err = tradeService.LimitSellByPercent(trade, percent)
		if err != nil {
			log.WithError(err).Error("Limit sell order failed.")
			handlers.WriteJsonResponse(w, http.StatusBadRequest, err.Error())
		}

		duration := time.Since(startTime)
		log.WithFields(log.Fields{
			"duration": duration,
			"symbol":   trade.State.Symbol,
		}).Debug("Sell order posted.")
	}
}

func limitSellByPriceHandler(tradeService *TradeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		if err := r.ParseForm(); err != nil {
			handlers.WriteBadRequestError(w)
			return
		}

		tradeId := vars["tradeId"]
		if tradeId == "" {
			handlers.WriteBadRequestError(w)
			return
		}

		if !handlers.RequireFormValue(w, r, "price") {
			return
		}

		price, err := strconv.ParseFloat(r.FormValue("price"), 64)
		if err != nil {
			handlers.WriteJsonError(w, http.StatusBadRequest,
				fmt.Sprintf("failed to parse price: %s: %v",
					r.FormValue("price"), err))
			return
		}

		trade := tradeService.FindTradeByLocalID(tradeId)
		if trade == nil {
			handlers.WriteJsonError(w, http.StatusNotFound, "")
			return
		}

		startTime := time.Now()

		if trade.State.Status == maker.TradeStatusPendingSell {
			log.Printf("Cancelling existing sell order.");
			tradeService.CancelSell(trade)
		}

		err = tradeService.LimitSellByPrice(trade, price)
		if err != nil {
			log.WithError(err).Error("Limit sell order failed.")
			handlers.WriteJsonResponse(w, http.StatusBadRequest, err.Error())
		}

		duration := time.Since(startTime)
		log.WithFields(log.Fields{
			"duration": duration,
			"symbol":   trade.State.Symbol,
		}).Debug("Sell order posted.")
	}
}

func marketSellHandler(tradeService *TradeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		tradeId := vars["tradeId"]
		if tradeId == "" {
			handlers.WriteBadRequestError(w)
			return
		}

		trade := tradeService.FindTradeByLocalID(tradeId)
		if trade == nil {
			handlers.WriteJsonError(w, http.StatusNotFound, "")
			return
		}

		if trade.State.Status == maker.TradeStatusPendingSell {
			log.Printf("Cancelling existing sell order.");
			tradeService.CancelSell(trade)
		}

		err := tradeService.MarketSell(trade, false)
		if err != nil {
			handlers.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		}
	}
}

func configHandler(w http.ResponseWriter, r *http.Request) {
	configFile := viper.ConfigFileUsed()
	buf, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Printf("error: failed to read %s: %v", configFile, err)
		return
	}
	yconf := map[interface{}]interface{}{}
	if err := yaml.Unmarshal(buf, &yconf); err != nil {
		log.Printf("error: failed to decode %s: %v", configFile, err)
		return
	}

	jconf := yaml2json(yconf)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(jconf); err != nil {
		log.WithError(err).Error("Failed to encode configuration.")
	}
}

func yaml2json(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = yaml2json(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = yaml2json(v)
		}
	}
	return i
}

func staticAssetHandler() http.HandlerFunc {
	static := packr.NewBox("../webapp/dist")
	fileServer := http.FileServer(static)
	return func(w http.ResponseWriter, r *http.Request) {
		if !static.Has(r.URL.Path) {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	}
}
