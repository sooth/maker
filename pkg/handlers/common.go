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
	"encoding/json"
	"fmt"
)

func WriteJsonResponse(w http.ResponseWriter, statusCode int, body interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	encoder := json.NewEncoder(w)
	return encoder.Encode(body)
}

func WriteJsonError(w http.ResponseWriter, statusCode int, message string) error {
	body := map[string]interface{}{
		"error":      true,
		"statusCode": statusCode,
	}
	if message != "" {
		body["message"] = message
	}
	return WriteJsonResponse(w, statusCode, body)
}

func RequireFormValue(w http.ResponseWriter, r *http.Request, field string) bool {
	if r.FormValue(field) == "" {
		WriteJsonError(w, http.StatusBadRequest, fmt.Sprintf("%s is required", field))
		return false
	}
	return true
}

func WriteBadRequestError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
}
