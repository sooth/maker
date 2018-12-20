package binanceex

import (
	"gitlab.com/crankykernel/cryptotrader/binance"
	"gitlab.com/crankykernel/maker/config"
)

func GetBinanceRestClient() *binance.RestClient {
	restClient := binance.NewAuthenticatedClient(
		config.GetString("binance.api.key"),
		config.GetString("binance.api.secret"))
	return restClient
}
