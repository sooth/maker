Remote Access
=============

The *Maker* application can be run on a server, however there are a
few extra steps required to enable remote access. To enable remote
access there are a few options available:

* SSH Forwarding
* Lets Encrypt TLS Certificate
* Self Signed Certificate
* Reverse Proxy

Option: SSH Forwarding
----------------------

The safest option to enable remote access is to use SSH forwarding.
This requires no special configuration, and only allows access to
those with SSH access to your server. You may want to enable
authentication if more than just you have access though.

Start *Maker* on the server::

  ./maker server --auth

The `--auth` is optional, but if you do use it, note the username and
password that are logged during startup.

Now on your device (laptop, phone, whatever) login to your server with
SSH and port forwarding. This will largely depend on your SSH client,
but using the standard SSH command line client this will look
something like::

  ssh -L6045:localhost:6045 <myserverip-hostname-or-ip>
  
Now point your browser at http://localhost:6045 and login if needed.

Option: Remote Access with Lets Encrypt TLS Certificate
-------------------------------------------------------

This is the best option if running on a remote server such as a VPS
and SSH forwarding is not available, or too inconvenient.

Requirements:

* A hostname that points to your servers IP address.
* A server you have root/administrative access to.

Once you have met the above requirements, enabling remote access with
a TLS certificate is very simple::

  ./maker server --letsencrypt --letsencrypt-hostname MYHOSTNAME --auth

As enabling Lets Encrypt requires remote access, authentication is
mandatory. If this is the first time enabling authentication, watch
the output for the *username* and *password*.

Option: Self Self Certificate
-----------------------------

*Maker* can also generate a self signed certificate for secure remote
access with TLS. However this might not be the most useful as Safari,
and in particular iOS devices will not connect to the required
websocket for a self signed certificate. But it could still be useful
for some.

TODO.

Reverse Proxy
-------------

TODO

Its All My Fault
----------------

Enabling remote access requires an extra command line option to
acknowledge that enabling remote access comes with risks, and its all
your fault it something goes wrong due to enabling remote access.

.. option:: --its-all-my-fault
