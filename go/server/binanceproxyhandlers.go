// Copyright (C) 2019 Cranky Kernel
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
	"github.com/gorilla/mux"
	"gitlab.com/crankykernel/maker/go/binanceex"
	"gitlab.com/crankykernel/maker/go/log"
	"net/http"
)

type BinanceProxyHandlers struct {
}

func NewBinanceProxyHandlers() *BinanceProxyHandlers {
	return &BinanceProxyHandlers{}
}

func (h *BinanceProxyHandlers) RegisterHandlers(router *mux.Router) {
	router.HandleFunc("/api/binance/proxy/getAccount", h.GetAccount)
}

func (h *BinanceProxyHandlers) GetAccount(w http.ResponseWriter, r *http.Request) {
	client := binanceex.GetBinanceRestClient()
	response, err := client.GetAccount()
	if err != nil {
		log.WithError(err).Errorf("Binance GetAccount request failed")
	}
	WriteJsonResponse(w, http.StatusOK, response)
}
