#!/usr/bin/env bash

set -e

UNAME=$(uname)

if [ $UNAME = Linux ]; then
	sudo apt-get install -qy pkg-config
fi
