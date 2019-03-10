Getting Started
===============

Binance Setup
-------------

Before starting, head over to Binance and setup your API key and
secret. The API key **must** have *trading* enabled, but it does not
require *withdrawals*, and you should probably have *withdrawals*
disabled.

It is also **highly recommended** to use **BNB** as a fee currency to
avoid `surprises
<https://maker.crankykernel.com/faq/#use-bnb-to-pay-fees-why-is-my-loss-so-large-so-quickly>`_.

Download the Maker Application
------------------------------

Download the application for your platform from:
https://maker.crankykernel.com/files/master/

At this time only builds for Linux/x86_64, Windows/x86_64 and
MacOS/x86_64 are provided.

Installation
------------

Windows
~~~~~~~

.. note:: At this time there is no real installer for Windows. The
	  application is just an exe file inside the zip archive.

* Extract the zip file and double click on ``maker.exe``. The zip file
  must be extracted, it will not work properly if running from the
  compressed folder.
* Alternatively you can run ``maker.exe`` from from the command prompt.

.. note:: You may get a warning about the app being untrusted. To run
	  **Maker** you are going to have to allow it to run.

Linux and MacOS
~~~~~~~~~~~~~~~

Extract the zip file and run::

  ./maker server

First Run
---------

If your browser didn't open to the *Maker* app by default, open your
browser to http://localhost:6045.

As this is the first time the app has run it will now ask for your
Binance API key and secret. Enter them in the form provided, **Test**,
then **Save**. You will only be given the option to **Save** your API
details if the authentication test passes.

.. note:: Maker is not a cloud based application and only connects to
	  Binance. Your API key and secret are only transmitted
	  between your computer and Binance in a secure manner.
