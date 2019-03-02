// The MIT License (MIT)
//
// Copyright (c) 2018-2019 Cranky Kernel
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use, copy,
// modify, merge, publish, distribute, sublicense, and/or sell copies
// of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS
// BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN
// ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package binanceex

import (
	"fmt"
	"github.com/crankykernel/binanceapi-go"
	"gitlab.com/crankykernel/maker/go/log"
	"sync"
)

type SymbolInfo struct {
	TickSize    float64
	StepSize    float64
	MinNotional float64
}

type ExchangeInfoService struct {
	Symbols map[string]SymbolInfo
	lock    sync.RWMutex
}

func NewExchangeInfoService() *ExchangeInfoService {
	return &ExchangeInfoService{
		Symbols: make(map[string]SymbolInfo),
	}
}

func (s *ExchangeInfoService) Update() error {
	exchangeInfo, err := binanceapi.NewRestClient().GetExchangeInfo()
	if err != nil {
		return err
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, symbol := range exchangeInfo.Symbols {
		symbolInfo := SymbolInfo{}
		for _, filter := range symbol.Filters {
			switch filter.FilterType {
			case "PRICE_FILTER":
				symbolInfo.TickSize = filter.TickSize
			case "MIN_NOTIONAL":
				symbolInfo.MinNotional = filter.MinNotional
			case "LOT_SIZE":
				symbolInfo.StepSize = filter.StepSize
			}
		}
		s.Symbols[symbol.Symbol] = symbolInfo
	}
	log.WithFields(log.Fields{
		"symbols": len(s.Symbols),
	}).Infof("Binance exchange info service updated")
	return nil
}

// GetSymbol returns the symbol info object for the requested symbol.
func (s *ExchangeInfoService) GetSymbol(symbol string) (info SymbolInfo, err error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	info, ok := s.Symbols[symbol]
	if !ok {
		return info, fmt.Errorf("symbol not found")
	}
	return info, nil
}

// GetTickSize returns the tick size for the requested symbol.
func (s *ExchangeInfoService) GetTickSize(symbol string) (float64, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	symbolInfo, ok := s.Symbols[symbol]
	if !ok {
		return 0, fmt.Errorf("symbol not found")
	}
	return symbolInfo.TickSize, nil
}

// GetMinNotional returns the minimum notional value for the requested symbol.
func (s *ExchangeInfoService) GetMinNotional(symbol string) (float64, error) {
	s.lock.RLock()
	defer s.lock.Unlock()
	symbolInfo, ok := s.Symbols[symbol]
	if !ok {
		return 0, fmt.Errorf("symbol not found")
	}
	return symbolInfo.MinNotional, nil
}

// GetStepSize returns the step size for the requested symbol.
func (s *ExchangeInfoService) GetStepSize(symbol string) (float64, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	symbolInfo, ok := s.Symbols[symbol]
	if !ok {
		return 0, fmt.Errorf("symbol not found")
	}
	return symbolInfo.StepSize, nil
}
