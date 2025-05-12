#!/bin/bash

export DBUS_SESSION_BUS_ADDRESS=`/bin/dbus-daemon --fork --config-file=/etc/dbus-1/session.conf --print-address`

exec "$@"
