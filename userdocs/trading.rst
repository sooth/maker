Trading
=======

Buy Price
---------

Best Bid
~~~~~~~~

Choosing **best bid** will place the buy order as a limit order for
the current highest bid price.

Best Ask
~~~~~~~~

Choosing **best ask** will place the buy order as a limit order for the
current lowest ask.

Last Price
~~~~~~~~~~

Chooing **last price** will place the buy order for the price of the last
trade.

Manual
~~~~~~

Choosing **manual** will place the buy order as a limit order for the
price specified.

Offset
~~~~~~

If using an automatic price source like **best bid**, **best ask**, or
**last price** you can set an offset that will adjust the actual buy
price by the number of ticks specified.

For example, choosing **best bid** and an offset of +1 *tick* should
place your order at the top of the order book.

.. note:: A **tick** is the minimum increment of price change allowed by
	  the exchange and varies with the asset. Something like DENTBTC may
	  have a tick size of 0.00000001, but ETHBTC has a tick size of
	  0.00000100.
	  
	  For DENTBTC setting the tick offset to +2 will adjust the price by
	  +0.00000002, but for ETHBTC the price will be adjusted by
	  +0.00000200.
	  
Trailing Profit
---------------

Enabling trailing profit sets up a sell when conditions are met. The
first condition is the % to **trigger** the trailing profit. When the
*last* price of the asset hits this level of profit the trailing
profit will activate, selling the asset when the price drops by the
percentage set in the **deviation**.

Trailing profit can be enabled or changed after the order is placed.

The sell order will be place as a **market order**.

Stop Loss
---------

Enable a stop loss to automatically exit your trade when the **last
price** is the specified % below your buy price.

Stop loss can be enabled or change after the order has been placed.

When the stop loss is triggered the trade will be sold with a **market
order**.

Limit Sell
----------

Setting a limit sell by % or price will place a limit sell order as
soon as the buy is filled.

This sell order can be canceled or changed after the order is made.

.. note:: This feature may also be known as **take profit**.
