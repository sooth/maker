# Maker

This is an application for creating and managing trades on crypto
currency exchanges (currently this only means Binance).

You run Maker on your own machine and access it with a web
browser. Your API keys and secrets are only sent between you and the
exchange servers.

## Supported Platforms

- Windows
- Linux
- MacOS

## Download

https://maker.crankykernel.com/files/master/

## Running

If using a .zip file download, first unzip then:

- On Linux: ./maker server
- On MacOS: Open the terminal and run "./maker server"
- On Windows: Double click "maker.exe"

Then using your web browser visit http://localhost:6045.

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
- This is **PRE BETA** software. Use at your own risk.

## Building

Before building _Maker_ you must install Go and Node.
- Node 10.15.0+
- Go 1.11.4+

The build process is known to work on Linux. It should also work on
MacOS, but probably won't work on Windows.

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

AGPL v3, with contributor agreement.

## ChangeLog

### unreleased
- By default hide API key/secret input, but add a checkbox to enable
  it to be shown.
- Periodically check the time difference between Binance and the Maker
  server and warn the user if it is too large and orders are likely to
  fail. https://gitlab.com/crankykernel/maker/issues/33.
- Disable cancel sell button when not
  applicable. https://gitlab.com/crankykernel/maker/issues/30
- Fix the amount of units to buy when using a manual price. The amount
  of units to buy was based on the last traded price, not the price
  entered. This is still an issue for bid/ask, but the error here will
  be much less and not really noticeable if only offseting the price
  by a few ticks. https://gitlab.com/crankykernel/maker/issues/34

### 0.3.1 - 2019-01-10
- Reload UI if backend version
  changes. https://gitlab.com/crankykernel/maker/issues/28
- Make the price offset in terms of ticks, no an actual
  value. https://gitlab.com/crankykernel/maker/issues/27
- Display 24 hour volume and price change for symbol being traded.

### 0.3.0 - 2019-01-03
- Update the web based user interface to Angular 7.
- Reconnect on loss of connection from a symbol trade stream.
- Fix trailing profit values on toggle on
  Firefox. https://gitlab.com/crankykernel/maker/issues/22
- Make limit order sell by value/percent buttons appear like radio
  buttons, as that is how they behave.
- Group the trailing profit, stop loss and limit sell into the trade
  card, so they are wrapped in the same border.
- Add a buy offset, which is an absolute value of the quote currency
  to adjust the buy price buy.
- (gitlab-ci) Add a MacOS build.
- (gitlab-ci) Remove 32 bit Windows build.
- More detailed logging to the log file.
- Each trade now contains a history of actions on the trade (may still
  be incomplete but its a start).
- Many UI tweaks and minor fixes/enhancements.

[Full Changelog](https://gitlab.com/crankykernel/maker/compare/0.2.1...0.3.0)

### 0.2.1

- Update for new Binance fee discount of 25%.

[Full Changelog](https://gitlab.com/crankykernel/maker/compare/0.2.0...0.2.1)

### 0.2.0

- When setting limit sell order when placing buy, allow an absolute
  price to be entered. https://gitlab.com/crankykernel/maker/issues/18
- Allow limit sell to be modified before buy is
  filled. https://gitlab.com/crankykernel/maker/issues/9
- Fixed issue where not all buy fills were being handled. In
  particular when a partial fill came after the fill.
- Rename "quick sell" to "limit sell" and add tooltip.

[Full Changelog](https://gitlab.com/crankykernel/maker/compare/0.1.0...0.2.0)

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
