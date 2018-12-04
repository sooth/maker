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
	"gitlab.com/crankykernel/cryptotrader/binance"
	"gitlab.com/crankykernel/maker/pkg/handlers"
	"gitlab.com/crankykernel/maker/pkg/log"
	"gitlab.com/crankykernel/maker/pkg/maker"
	"io/ioutil"
	"net/http"
)

func PostBuyHandler(tradeService *TradeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		log.Debugf("Received buy order request: %v", requestBody.AsJson())

		commonLogFields := log.Fields{
			"symbol": requestBody.Symbol,
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

		// Validate limit sell.
		if requestBody.LimitSellEnabled {
			switch requestBody.LimitSellType {
			case maker.LimitSellTypePercent:
			case maker.LimitSellTypePrice:
			default:
				handlers.WriteJsonError(w, http.StatusBadRequest,
					fmt.Sprintf("limit sell type invalid or not set"))
				return
			}
		}

		params.Symbol = requestBody.Symbol
		params.Quantity = requestBody.Quantity

		orderId, err := tradeService.MakeOrderID()
		if err != nil {
			log.WithFields(commonLogFields).WithError(err).Errorf("Failed to create order ID.")
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
			params.Price = requestBody.Price
		default:
			params.Price, err = buyService.GetPrice(params.Symbol, requestBody.PriceSource)
			if err != nil {
				log.WithError(err).WithFields(commonLogFields).WithFields(log.Fields{
					"priceSource": requestBody.PriceSource,
				}).Error("Failed to get buy price.")
				handlers.WriteJsonError(w, http.StatusInternalServerError,
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
			if requestBody.LimitSellType == maker.LimitSellTypePercent {
				log.WithFields(commonLogFields).Infof("Setting limit sell at %f percent.",
					requestBody.LimitSellPercent)
				trade.SetLimitSellByPercent(requestBody.LimitSellPercent)
			} else if requestBody.LimitSellType == maker.LimitSellTypePrice {
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
