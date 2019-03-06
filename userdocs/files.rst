Files and Directories
=====================

Data Directory
--------------

The *data directory* is where *Maker* stores all its files including
configuration, log files and trade database.

Windows
    %APPDATA%/MakerTradingTool

Linux
    $HOME/.makertradingtool

MacOS
    $HOME/.makertradingtool

.. note:: You may have installed *Maker* before this directory was in
          use, in which case the data files shown below will reside
          alongside your maker.exe file.

Files
-----

maker.yaml
    The configuration file. This file contains your exchange API
    connection detail, authentication information and other
    configuration data.

maker.db
    This is an SQLite database that contains all the trades including
    active (open) trades as well as a ll closed and cancel trades.

maker.log
    The log file. This can be useful when requesting support for an
    issue. While this file will contain details of trades it does not
    contain exchange API information such as your key or secret.

maker.pem
    If TLS support has been enabled this will contain the self sign
    TLS certificate.
