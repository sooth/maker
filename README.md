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

## Building

Before building _Maker_ you must install Go and Node.
- Node 8.11.3+
- Go 1.10.3+

A Makefile is used to complete some steps of the build, so install
_make_ as well.

1. From the top of the source tree run:

		make install-deps

	This command will:
	- In webapp into npm dependencies: `npm install`.
	- In the top level directory, install the Go dependencies.

2. In the top level directory run:

		make

	This will produce the *maker* binary in the current directory with
    the web application resources bundled into it.

## License

AGPL v3.

## ChangeLog

### 0.2.0

- When setting limit sell order when placing buy, allow an absolute
  price to be entered. https://gitlab.com/crankykernel/maker/issues/18
- Allow limit sell to be modified before buy is
  filled. https://gitlab.com/crankykernel/maker/issues/9
- Fixed issue where not all buy fills were being handled. In
  particular when a partial fill came after the fill.
- Rename "quick sell" to "limit sell" and add tooltip.

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
