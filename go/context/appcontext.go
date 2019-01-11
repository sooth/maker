package context

import (
	"gitlab.com/crankykernel/maker/binanceex"
	"gitlab.com/crankykernel/maker/tradeservice"
)

type ApplicationContext struct {
	TradeService              *tradeservice.TradeService
	//BinanceTradeStreamManager *binanceex.TradeStreamManager
	BinanceTradeStreamManager *binanceex.TradeStreamManager
	BinanceUserDataStream     *binanceex.BinanceUserDataStream
	OpenBrowser               bool
}
