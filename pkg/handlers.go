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
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"encoding/json"
	"github.com/crankykernel/maker/pkg/log"
	"strconv"
	"github.com/crankykernel/cryptotrader/binance"
	"fmt"
	"io"
	"github.com/gobuffalo/packr"
)

func writeJsonResponse(w http.ResponseWriter, statusCode int, body interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	encoder := json.NewEncoder(w)
	return encoder.Encode(body)
}

func writeBadRequestError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
}

func writeJsonError(w http.ResponseWriter, statusCode int, message string) error {
	body := map[string]interface{}{
		"error":      true,
		"statusCode": statusCode,
	}
	if message != "" {
		body["message"] = message
	}
	return writeJsonResponse(w, statusCode, body)
}

func archiveTradeHandler(tradeService *TradeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		tradeId := vars["tradeId"]
		if tradeId == "" {
			writeJsonError(w, http.StatusBadRequest, "tradeId required")
			return
		}

		logFields := log.Fields{
			"tradeId": tradeId,
		}

		trade := tradeService.FindTradeByLocalID(tradeId)
		if trade == nil {
			log.WithFields(logFields).
				Warn("Failed to archive trade, tradeId not found.")
			writeJsonError(w, http.StatusNotFound, "trade not found")
			return
		}

		if err := tradeService.ArchiveTrade(trade); err != nil {
			log.WithFields(logFields).
				WithError(err).Error("Failed to archive trade.")
			writeJsonError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.WithFields(logFields).Info("Trade archived.")
	}
}

func updateTradeStopLossSettingsHandler(tradeService *TradeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		if err = r.ParseForm(); err != nil {
			writeBadRequestError(w)
			return
		}

		var tradeId string
		var enable bool
		var percent float64

		vars := mux.Vars(r)
		tradeId = vars["tradeId"]
		if tradeId == "" {
			writeBadRequestError(w)
			return
		}

		if enable, err = strconv.ParseBool(r.FormValue("enable")); err != nil {
			writeBadRequestError(w)
			return
		}
		if percent, err = strconv.ParseFloat(r.FormValue("percent"), 64); err != nil {
			writeBadRequestError(w)
			return
		}

		trade := tradeService.FindTradeByLocalID(tradeId)
		if trade == nil {
			log.Printf("Failed to find trade with ID %s.", tradeId)
			writeJsonError(w, http.StatusNotFound, "")
		}

		log.Printf("Updating stop loss for trade %s: enable=%v; percent=%v",
			tradeId, enable, percent)
		tradeService.UpdateStopLoss(trade, enable, percent)
	}
}

func updateTradeTrailingStopSettingsHandler(tradeService *TradeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		if err = r.ParseForm(); err != nil {
			writeBadRequestError(w)
			return
		}

		var tradeId string
		var enable bool
		var percent float64
		var deviation float64

		vars := mux.Vars(r)
		tradeId = vars["tradeId"]
		if tradeId == "" {
			writeBadRequestError(w)
			return
		}

		if enable, err = strconv.ParseBool(r.FormValue("enable")); err != nil {
			writeBadRequestError(w)
			return
		}
		if percent, err = strconv.ParseFloat(r.FormValue("percent"), 64); err != nil {
			writeBadRequestError(w)
			return
		}
		if deviation, err = strconv.ParseFloat(r.FormValue("deviation"), 64); err != nil {
			writeBadRequestError(w)
			return
		}

		trade := tradeService.FindTradeByLocalID(tradeId)
		if trade == nil {
			log.Printf("Failed to find trade with ID %s.", tradeId)
			writeJsonError(w, http.StatusNotFound, "")
		}

		tradeService.UpdateTrailingStop(trade, enable, percent, deviation)
	}
}

func deleteBuyHandler(tradeService *TradeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			writeBadRequestError(w)
			return
		}

		tradeId := r.FormValue("trade_id")
		if tradeId == "" {
			writeBadRequestError(w)
		}

		trade := tradeService.FindTradeByLocalID(tradeId)
		if trade == nil {
			log.WithFields(log.Fields{
				"tradeId": tradeId,
			}).Warnf("Failed to cancel buy order. Trade ID not found.")
			writeBadRequestError(w)
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

		writeJsonResponse(w, http.StatusOK, response)
	}
}

func deleteSellHandler(tradeService *TradeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		tradeId := r.FormValue("trade_id")

		if tradeId == "" {
			writeBadRequestError(w)
			return
		}

		trade := tradeService.FindTradeByLocalID(tradeId)
		if trade == nil {
			log.WithFields(log.Fields{
				"tradeId": tradeId,
			}).Warnf("Failed to cancel sell order. No trade found for ID.")
			writeBadRequestError(w)
			return
		}

		log.WithFields(log.Fields{
			"symbol":      trade.State.Symbol,
			"tradeId":     trade.State.LocalID,
			"sellOrderId": trade.State.SellOrderId,
		}).Infof("Cancelling sell order.")

		response, err := getBinanceRestClient().CancelOrder(trade.State.Symbol,
			trade.State.SellOrderId)
		if err != nil {
			log.Printf("error: failed to post order: %v", err)
			return
		}

		writeJsonResponse(w, http.StatusOK, response)
	}
}

func postBuyHandler(tradeService *TradeService) http.HandlerFunc {

	type BuyOrderResponse struct {
		TradeID string `json:"trade_id""`
	}

	type BuyOrderRequestBody struct {
		LimitSellEnabled bool    `json:"limitSellEnabled"`
		LimitSellPercent float64 `json:"limitSellPercent"`

		StopLossEnabled bool    `json:"stopLossEnabled"`
		StopLossPercent float64 `json:"stopLossPercent"`

		TrailingStopEnabled   bool    `json:"trailingStopEnabled"`
		TrailingStopPercent   float64 `json:"trailingStopPercent"`
		TrailingStopDeviation float64 `json:"trailingStopDeviation"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		params := binance.OrderParameters{
			Side: binance.OrderSideBuy,
		}

		for key := range r.Form {
			var err error
			var val = r.FormValue(key)
			switch key {
			case "price":
				params.Price, err = strconv.ParseFloat(val, 64)
			case "quantity":
				params.Quantity, err = strconv.ParseFloat(val, 64)
			case "symbol":
				params.Symbol = val
			case "type":
				params.Type = binance.OrderType(val)
			case "timeInForce":
				params.TimeInForce = binance.TimeInForce(val)
			}
			if err != nil {
				log.Printf("error: failed to convert order: %v", err)
				return
			}
		}

		orderId, err := tradeService.MakeOrderID()
		if err != nil {
			log.WithError(err).Errorf("Failed to create order ID.")
			writeJsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		params.NewClientOrderId = orderId

		var requestBody BuyOrderRequestBody
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&requestBody); err != nil {
			log.Printf("error: failed to decode request body: %v", err)
			writeBadRequestError(w)
			return
		}

		trade := NewTrade()
		trade.State.Symbol = params.Symbol
		trade.AddClientOrderID(params.NewClientOrderId)

		if requestBody.StopLossEnabled {
			trade.SetStopLoss(requestBody.StopLossEnabled,
				requestBody.StopLossPercent)
		}

		if requestBody.LimitSellEnabled {
			trade.SetLimitSell(requestBody.LimitSellEnabled,
				requestBody.LimitSellPercent)
		}

		if requestBody.TrailingStopEnabled {
			trade.SetTrailingStop(requestBody.TrailingStopEnabled,
				requestBody.TrailingStopPercent,
				requestBody.TrailingStopDeviation)
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
			log.Printf("error: failed to post order: %v", err)
			switch err := err.(type) {
			case *binance.RestApiError:
				for key, val := range response.Header {
					w.Header()[key] = val
				}
				w.WriteHeader(response.StatusCode)
				w.Write(err.Body)
			default:
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("%v", err)))
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

		writeJsonResponse(w, http.StatusOK, BuyOrderResponse{
			TradeID: tradeId,
		})
	}
}

func postSellHandler(tradeService *TradeService) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		params := binance.OrderParameters{
			Side: binance.OrderSideSell,
		}

		var tradeId string

		for key := range r.Form {
			var err error
			var val = r.FormValue(key)
			switch key {
			case "trade_id":
				tradeId = val
			case "price":
				params.Price, err = strconv.ParseFloat(val, 64)
			case "quantity":
				params.Quantity, err = strconv.ParseFloat(val, 64)
			case "symbol":
				params.Symbol = val
			case "type":
				params.Type = binance.OrderType(val)
			case "timeInForce":
				params.TimeInForce = binance.TimeInForce(val)
			}
			if err != nil {
				log.Printf("error: failed to convert order: %v", err)
				return
			}
		}

		orderId, err := tradeService.MakeOrderID()
		if err != nil {
			log.WithError(err).Errorf("Failed to create order ID.")
			writeJsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		params.NewClientOrderId = orderId

		if tradeId != "" {
			trade := tradeService.FindTradeByLocalID(tradeId)
			if trade == nil {
				log.Printf("error: failed to find trade with id %s", tradeId)
			} else {
				tradeService.AddClientOrderId(trade, params.NewClientOrderId, false)
			}
		} else {
			log.Printf("error: no trade id provided")
			writeBadRequestError(w)
			return
		}

		log.WithFields(log.Fields{
			"tradeId":       tradeId,
			"symbol":        params.Symbol,
			"price":         params.Price,
			"quantity":      params.Quantity,
			"type":          params.Type,
			"clientOrderId": params.NewClientOrderId,
		}).Infof("Posting SELL order for %s", params.Symbol)

		response, err := getBinanceRestClient().PostOrder(params)
		if err != nil {
			log.Printf("error: failed to post order: %v", err)
			switch err := err.(type) {
			case *binance.RestApiError:
				for key, val := range response.Header {
					w.Header()[key] = val
				}
				w.WriteHeader(response.StatusCode)
				w.Write(err.Body)
			default:
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("%v", err)))
			}
			return
		}

		for key, val := range response.Header {
			w.Header()[key] = val
		}
		w.WriteHeader(response.StatusCode)
		io.Copy(w, response.Body)
	}
}

func limitSellHandler(tradeService *TradeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		if err := r.ParseForm(); err != nil {
			writeBadRequestError(w)
			return
		}

		tradeId := vars["tradeId"]
		if tradeId == "" {
			writeBadRequestError(w)
			return
		}

		percent, err := strconv.ParseFloat(r.FormValue("percent"), 64)
		if err != nil {
			writeBadRequestError(w)
			return
		}

		trade := tradeService.FindTradeByLocalID(tradeId)
		if trade == nil {
			writeJsonError(w, http.StatusNotFound, "")
			return
		}

		if trade.State.Status == TradeStatusPendingSell {
			log.Printf("Cancelling existing sell order.");
			tradeService.CancelSell(trade)
		}

		tradeService.DoLimitSell(trade, percent)
	}
}

func marketSellHandler(tradeService *TradeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		tradeId := vars["tradeId"]
		if tradeId == "" {
			writeBadRequestError(w)
			return
		}

		trade := tradeService.FindTradeByLocalID(tradeId)
		if trade == nil {
			writeJsonError(w, http.StatusNotFound, "")
			return
		}

		if trade.State.Status == TradeStatusPendingSell {
			log.Printf("Cancelling existing sell order.");
			tradeService.CancelSell(trade)
		}

		err := tradeService.MarketSell(trade, false)
		if err != nil {
			writeJsonError(w, http.StatusInternalServerError, err.Error())
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
