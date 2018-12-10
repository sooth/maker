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
	"encoding/json"
	"fmt"
	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"gitlab.com/crankykernel/maker/handlers"
	"gitlab.com/crankykernel/maker/log"
	"gitlab.com/crankykernel/maker/types"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
	"gitlab.com/crankykernel/maker/db"
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

func DeleteSellHandler(tradeService *TradeService) http.HandlerFunc {
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

		switch trade.State.Status {
		case types.TradeStatusNew:
			fallthrough
		case types.TradeStatusPendingBuy:
			trade.State.LimitSell.Enabled = false
			db.DbUpdateTrade(trade)
			tradeService.BroadcastTradeUpdate(trade)
			return
		}

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

		switch trade.State.Status {
		case types.TradeStatusNew:
			fallthrough
		case types.TradeStatusPendingBuy:
			trade.SetLimitSellByPercent(percent)
			db.DbUpdateTrade(trade)
			tradeService.BroadcastTradeUpdate(trade)
			log.WithFields(log.Fields{
				"symbol":  trade.State.Symbol,
				"tradeId": trade.State.TradeID,
				"percent": percent,
			}).Info("Updated limit sell on buy.")
			return
		}

		startTime := time.Now()

		if trade.State.Status == types.TradeStatusPendingSell {
			log.Printf("Cancelling existing sell order.")
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

		switch trade.State.Status {
		case types.TradeStatusNew:
			fallthrough
		case types.TradeStatusPendingBuy:
			trade.SetLimitSellByPrice(price)
			db.DbUpdateTrade(trade)
			tradeService.BroadcastTradeUpdate(trade)
			log.WithFields(log.Fields{
				"symbol":  trade.State.Symbol,
				"tradeId": trade.State.TradeID,
				"price":   price,
			}).Info("Updated limit sell on buy.")
			return
		}

		startTime := time.Now()

		if trade.State.Status == types.TradeStatusPendingSell {
			log.Printf("Cancelling existing sell order.")
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

		if trade.State.Status == types.TradeStatusPendingSell {
			log.Printf("Cancelling existing sell order.")
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
	static := packr.NewBox("../../webapp/dist")
	fileServer := http.FileServer(static)
	return func(w http.ResponseWriter, r *http.Request) {
		if !static.Has(r.URL.Path) {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	}
}
