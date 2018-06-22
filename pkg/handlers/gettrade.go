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

package handlers

import (
	"net/http"
	"github.com/gorilla/mux"
	"github.com/crankykernel/maker/pkg/db"
	"github.com/crankykernel/maker/pkg/log"
)

func GetTrade(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tradeId := vars["tradeId"]
	trade, err := db.DbGetTradeByID(tradeId)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"tradeId": tradeId,
		}).Warn("Failed to find trade by ID.")
		WriteJsonResponse(w, http.StatusNotFound, "trade not found")
		return
	}
	WriteJsonResponse(w, http.StatusOK, trade)
}