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
