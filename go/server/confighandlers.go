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
	"gitlab.com/crankykernel/cryptotrader/binance"
	"gitlab.com/crankykernel/maker/config"
	"gitlab.com/crankykernel/maker/log"
	"net/http"
)

func SavePreferencesHandler(w http.ResponseWriter, r *http.Request) {
	type preferenceConfig struct {
		BalancePercents string `json:"balancePercents"`
	}

	var request preferenceConfig
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		log.WithFields(log.Fields{
			"path":   r.URL.Path,
			"method": r.Method,
		}).WithError(err).Errorf("Failed to decode Binance configuration.")
		WriteJsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	config.Set("preferences.balance.percents", request.BalancePercents)
	config.WriteConfig()
}

func SaveBinanceConfigHandler(w http.ResponseWriter, r *http.Request) {
	type binanceApiConfiguration struct {
		ApiKey    string `json:"key"`
		ApiSecret string `json:"secret"`
	}

	var request binanceApiConfiguration
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		log.WithFields(log.Fields{
			"path":   r.URL.Path,
			"method": r.Method,
		}).WithError(err).Errorf("Failed to decode Binance configuration.")
		WriteJsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	config.Set("binance.api.key", request.ApiKey)
	config.Set("binance.api.secret", request.ApiSecret)
	config.WriteConfig()
}

func BinanceTestHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		WriteJsonError(w, http.StatusBadRequest, "failed to parse form data")
		return
	}

	binanceApiKey := r.FormValue("binance.api.key")
	if binanceApiKey == "" {
		WriteJsonError(w, http.StatusBadRequest, "missing binance.api.key")
		return
	}
	binanceApiSecret := r.FormValue("binance.api.secret")
	if binanceApiSecret == "" {
		WriteJsonError(w, http.StatusBadRequest, "missing binance.api.secret")
		return
	}

	client := binance.NewAuthenticatedClient(binanceApiKey, binanceApiSecret)
	_, err := client.GetAccount()
	if err != nil {
		log.WithError(err).Warn("Binance account authentication test failed.")
		WriteJsonResponse(w, http.StatusOK, map[string]interface{}{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}
