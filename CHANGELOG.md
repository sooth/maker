## ChangeLog

### unreleased
- New command line option to set the data directory where the
  maker.yaml, maker.db and log files are stored.  This replaces the
  options to set the config filename and log
  filename. https://gitlab.com/crankykernel/maker/issues/38
- Store the configuration and database in a fixed location to avoid
  having to move the files over upgrades. On Linux this directory is
  ~/.makertradingtool and on Windows it is
  %appdata%\MakerTradingTool. It can be modified using the command
  line option above to change the data
  directory. https://gitlab.com/crankykernel/maker/issues/31
- Filter display of active trades to all, open closed.
  https://gitlab.com/crankykernel/maker/issues/32
- Add TLS support. TLS support can be enabled with the `--tls` command
  line option, or by binding to non-localhost.
  https://gitlab.com/crankykernel/maker/issues/3
- Add authentication support. The `--auth` command line option is used
  to enable authentication and will auto generate a strong password.
  https://gitlab.com/crankykernel/maker/issues/1
- Move AccountInfo request (Binance) from the UI to the server so the
  UI doesn't have to make any authenticated requests to
  Binance. Solves the issue where the server on a VPS may have good
  time sync, but the browser machine
  doesn't. https://gitlab.com/crankykernel/maker/issues/52
- Provide visiable health status in the
  UI. https://gitlab.com/crankykernel/maker/issues/42
- Add simple Binance balance view.

[Full Changelog](https://gitlab.com/crankykernel/maker/compare/0.3.2...master)

### 0.3.2 - 2019-02-01
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
- Add a "Check for Update" button to the "About" page. Note that this
  will "phone home" to a specific URL that returns details about the
  latest versions. https://gitlab.com/crankykernel/maker/issues/25
- On MacOS, if no arguments passed assume it was double clicked on and
  start in server mode and attempt to launch the
  browser. https://gitlab.com/crankykernel/maker/issues/37

[Full Changelog](https://gitlab.com/crankykernel/maker/compare/0.3.1...0.3.2)

### 0.3.1 - 2019-01-10
- Reload UI if backend version
  changes. https://gitlab.com/crankykernel/maker/issues/28
- Make the price offset in terms of ticks, no an actual
  value. https://gitlab.com/crankykernel/maker/issues/27
- Display 24 hour volume and price change for symbol being traded.

[Full Changelog](https://gitlab.com/crankykernel/maker/compare/0.3.0...0.3.1)

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
