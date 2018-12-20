package context

import (
	"gitlab.com/crankykernel/maker/binanceex"
	"gitlab.com/crankykernel/maker/tradeservice"
)

type ApplicationContext struct {
	TradeService          *tradeservice.TradeService
	BinanceStreamManager  *binanceex.BinanceStreamManager
	BinanceUserDataStream *binanceex.BinanceUserDataStream
	OpenBrowser           bool
}
