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

import "fmt"
import (
	"github.com/crankykernel/cryptotrader/binance"
	"github.com/crankykernel/maker/pkg/maker"
)

type BinanceBuyService struct {
	anonymousClient *binance.RestClient
}

func NewBinanceBuyService() *BinanceBuyService {
	return &BinanceBuyService{
		anonymousClient: binance.NewAnonymousClient(),
	}
}

func (s *BinanceBuyService) GetPrice(symbol string, priceSource maker.PriceSource) (float64, error) {
	if priceSource == maker.PriceSourceLast {
		ticker, err := s.anonymousClient.GetPriceTicker(symbol)
		if err != nil {
			return 0, err
		}
		return ticker.Price, nil
	} else if priceSource == maker.PriceSourceBaskAsk || priceSource == maker.PriceSourceBestBid {
		ticker, err := s.anonymousClient.GetOrderBookTicker(symbol)
		if err != nil {
			return 0, err
		}
		switch priceSource {
		case maker.PriceSourceBestBid:
			return ticker.BidPrice, nil
		case maker.PriceSourceBaskAsk:
			return ticker.AskPrice, nil
		}
	}
	return 0, fmt.Errorf("unknown price source: %s", priceSource)
}
