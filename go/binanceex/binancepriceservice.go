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

package binanceex

import (
	"fmt"
	"gitlab.com/crankykernel/cryptotrader/binance"
	"gitlab.com/crankykernel/maker/go/log"
	"gitlab.com/crankykernel/maker/go/types"
	"gitlab.com/crankykernel/maker/go/util"
)

type BinancePriceService struct {
	anonymousClient     *binance.RestClient
	exchangeInfoService *binance.ExchangeInfoService
}

func NewBinancePriceService(exchangeInfoService *binance.ExchangeInfoService) *BinancePriceService {
	return &BinancePriceService{
		anonymousClient:     binance.NewAnonymousClient(),
		exchangeInfoService: exchangeInfoService,
	}
}

// GetLastPrice gets the most current close price from Binance using the REST
// API.
func (s *BinancePriceService) GetLastPrice(symbol string) (float64, error) {
	ticker, err := s.anonymousClient.GetPriceTicker(symbol)
	if err != nil {
		return 0, err
	}
	return ticker.Price, nil
}

// GetBestBidPrice gets the most current best bid price from Binance using
// the REST API.
func (s *BinancePriceService) GetBestBidPrice(symbol string) (float64, error) {
	ticker, err := s.anonymousClient.GetOrderBookTicker(symbol)
	if err != nil {
		return 0, err
	}
	return ticker.BidPrice, nil
}

// GetBestBidPrice gets the most current best bid price from Binance using
// the REST API.
func (s *BinancePriceService) GetBestAskPrice(symbol string) (float64, error) {
	ticker, err := s.anonymousClient.GetOrderBookTicker(symbol)
	if err != nil {
		return 0, err
	}
	return ticker.AskPrice, nil
}

func (s *BinancePriceService) AdjustPriceByTicks(symbol string, price float64, ticks int64) float64 {
	tickSize, err := s.exchangeInfoService.GetTickSize(symbol)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"symbol": symbol,
		}).Errorf("Failed to lookup tick size")
	}
	return util.Round8(price + (tickSize * float64(ticks)))
}

func (s *BinancePriceService) GetPrice(symbol string, priceSource types.PriceSource) (float64, error) {
	switch priceSource {
	case types.PriceSourceLast:
		return s.GetLastPrice(symbol)
	case types.PriceSourceBestBid:
		return s.GetBestBidPrice(symbol)
	case types.PriceSourceBestAsk:
		return s.GetBestAskPrice(symbol)
	default:
		return 0, fmt.Errorf("unknown price source: %s", priceSource)
	}
}
