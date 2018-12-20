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
	"encoding/json"
	"fmt"
	"gitlab.com/crankykernel/cryptotrader/binance"
	"gitlab.com/crankykernel/maker/binanceex"
	"gitlab.com/crankykernel/maker/log"
	"gitlab.com/crankykernel/maker/tradeservice"
	"gitlab.com/crankykernel/maker/types"
	"io/ioutil"
	"net/http"
	"time"
)

type LimitSellRequest struct {
	Enabled bool                `json:"enabled"`
	Type    types.LimitSellType `json:"type"`
	Percent float64             `json:"percent"`
	Price   float64             `json:"price"`
}

type BuyOrderRequest struct {
	Symbol                  string              `json:"symbol"`
	Quantity                float64             `json:"quantity"`
	PriceSource             types.PriceSource   `json:"priceSource"`
	LimitSellEnabled        bool                `json:"limitSellEnabled"`
	LimitSellType           types.LimitSellType `json:"limitSellType"`
	LimitSellPercent        float64             `json:"limitSellPercent"`
	LimitSellPrice          float64             `json:"limitSellPrice"`
	StopLossEnabled         bool                `json:"stopLossEnabled"`
	StopLossPercent         float64             `json:"stopLossPercent"`
	TrailingProfitEnabled   bool                `json:"trailingProfitEnabled"`
	TrailingProfitPercent   float64             `json:"trailingProfitPercent"`
	TrailingProfitDeviation float64             `json:"trailingProfitDeviation"`
	Price                   float64             `json:"price"`
}

func (r *BuyOrderRequest) AsJson() string {
	bytes, err := json.Marshal(r)
	if err != nil {
		return ""
	}
	return string(bytes)
}

type BuyOrderResponse struct {
	TradeID string `json:"trade_id""`
}

func PostBuyHandler(tradeService *tradeservice.TradeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := binance.OrderParameters{
			Side:        binance.OrderSideBuy,
			Type:        binance.OrderTypeLimit,
			TimeInForce: binance.TimeInForceGTC,
		}

		log.Printf("params: %v", log.ToJson(params))

		var requestBody BuyOrderRequest
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&requestBody); err != nil {
			log.Printf("error: failed to decode request body: %v", err)
			WriteBadRequestError(w)
			return
		}

		log.Debugf("Received buy order request: %v", requestBody.AsJson())

		commonLogFields := log.Fields{
			"symbol": requestBody.Symbol,
		}

		// Validate price source.
		switch requestBody.PriceSource {
		case types.PriceSourceLast:
		case types.PriceSourceBestBid:
		case types.PriceSourceBestAsk:
		case types.PriceSourceManual:
		case "":
			WriteJsonError(w, http.StatusBadRequest, "missing required parameter: priceSource")
			return
		default:
			WriteJsonError(w, http.StatusBadRequest,
				fmt.Sprintf("invalid value for priceSource: %v", requestBody.PriceSource))
			return
		}

		// Validate limit sell.
		if requestBody.LimitSellEnabled {
			switch requestBody.LimitSellType {
			case types.LimitSellTypePercent:
			case types.LimitSellTypePrice:
			default:
				WriteJsonError(w, http.StatusBadRequest,
					fmt.Sprintf("limit sell type invalid or not set"))
				return
			}
		}

		params.Symbol = requestBody.Symbol
		params.Quantity = requestBody.Quantity

		orderId, err := tradeService.MakeOrderID()
		if err != nil {
			log.WithFields(commonLogFields).WithError(err).Errorf("Failed to create order ID.")
			WriteJsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		params.NewClientOrderId = orderId

		trade := types.NewTrade()
		trade.AddHistory(types.HistoryEntry{
			Timestamp: time.Now(),
			Type: types.Created,
			Fields: requestBody,
		})
		trade.State.Symbol = params.Symbol
		trade.AddClientOrderID(params.NewClientOrderId)

		buyService := binanceex.NewBinancePriceService()

		switch requestBody.PriceSource {
		case types.PriceSourceManual:
			params.Price = requestBody.Price
		default:
			params.Price, err = buyService.GetPrice(params.Symbol, requestBody.PriceSource)
			if err != nil {
				log.WithError(err).WithFields(commonLogFields).WithFields(log.Fields{
					"priceSource": requestBody.PriceSource,
				}).Error("Failed to get buy price.")
				WriteJsonError(w, http.StatusInternalServerError,
					fmt.Sprintf("Failed to get price: %v", err))
				return
			}
		}

		if requestBody.StopLossEnabled {
			trade.SetStopLoss(requestBody.StopLossEnabled,
				requestBody.StopLossPercent)
		}

		if requestBody.TrailingProfitEnabled {
			trade.SetTrailingProfit(requestBody.TrailingProfitEnabled,
				requestBody.TrailingProfitPercent,
				requestBody.TrailingProfitDeviation)
		}

		tradeId := tradeService.AddNewTrade(trade)
		commonLogFields["tradeId"] = tradeId;

		if requestBody.LimitSellEnabled {
			if requestBody.LimitSellType == types.LimitSellTypePercent {
				log.WithFields(commonLogFields).Infof("Setting limit sell at %f percent.",
					requestBody.LimitSellPercent)
				trade.SetLimitSellByPercent(requestBody.LimitSellPercent)
			} else if requestBody.LimitSellType == types.LimitSellTypePrice {
				log.WithFields(commonLogFields).Infof("Setting limit sell at price %f.",
					requestBody.LimitSellPrice)
				trade.SetLimitSellByPrice(requestBody.LimitSellPrice)
			}
		}

		log.WithFields(commonLogFields).WithFields(log.Fields{
			"type":                    params.Type,
			"price":                   params.Price,
			"quantity":                params.Quantity,
			"clientOrderId":           params.NewClientOrderId,
			"priceSource":             requestBody.PriceSource,
			"limitSellEnabled":        requestBody.LimitSellEnabled,
			"limitSellType":           requestBody.LimitSellType,
			"limitSellPercent":        requestBody.LimitSellPercent,
			"limitSellPrice":          requestBody.LimitSellPrice,
			"stopLossEnabled":         requestBody.StopLossEnabled,
			"stopLossPercent":         requestBody.StopLossPercent,
			"trailingProfitEnabled":   requestBody.TrailingProfitEnabled,
			"trailingProfitPercent":   requestBody.TrailingProfitPercent,
			"trailingProfitDeviation": requestBody.TrailingProfitDeviation,
		}).Infof("Posting BUY order for %s", params.Symbol)

		response, err := binanceex.GetBinanceRestClient().PostOrder(params)
		if err != nil {
			log.WithError(err).
				Errorf("Failed to post buy order.")
			switch err := err.(type) {
			case *binance.RestApiError:
				log.Debugf("Forwarding Binance error repsonse.")
				w.WriteHeader(response.StatusCode)
				w.Write(err.Body)
			default:
				WriteJsonResponse(w, http.StatusInternalServerError,
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

		WriteJsonResponse(w, http.StatusOK, BuyOrderResponse{
			TradeID: tradeId,
		})
	}
}
