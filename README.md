# Maker

This is an application for creating and managing trades on crypto
currency exchanges (currently this only means Binance).

You run Maker on your own machine and access it with a web
browser. Your API keys and secrets are only sent between you and the
exchange servers.

## Supported Platforms
-------------------

- Windows
- Linux

## Download

https://gitlab.com/crankykernel/maker/-/jobs/artifacts/master/browse?job=build

## Supported Exchanges

- Binance
	
## Features

- Choose the amount to buy based on percentage of your balance.
- Support for all quote currencies on Binance.
- Stop loss.
- Trailing profit/stop.
- Quick limit sell - up fill of your buy order automatically place a
  limit sell for a specified percent.

## Warnings

- Most testing has currently been done on BTC pairs.
- The application must remain running for trailing profit and stop
  loss to execute.
- This is PRE BETA software. Use at your own risk.

## License

AGPL v3.

## ChangeLog

### 0.1.1

Tagging what has been done as 0.1.1 as its low risk modifications, and
the 0.1.0 binary build was broken.

#### Added
- Allow a trade to be sold at a fixed
  price. https://gitlab.com/crankykernel/maker/issues/14
- Require confirmation to market sell.
- Require confirmation to abandon a trade.
- Allow a trade to be bought at a fixed
  price. https://gitlab.com/crankykernel/maker/issues/14

[Full Changelog](https://gitlab.com/crankykernel/maker/compare/0.1.0...master)

[Download](https://gitlab.com/crankykernel/maker/-/jobs/artifacts/master/browse?job=build)

### 0.1.0
- Initial release.

[Full Changelog](https://gitlab.com/crankykernel/maker/commits/0.1.0)

[Download](https://gitlab.com/crankykernel/maker/-/jobs/artifacts/0.1.0/browse?job=build)
