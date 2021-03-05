#!/usr/bin/env sh

openssl req -new \
	-newkey rsa:4096 -x509 \
	-sha256 -days 3650 \
	-nodes -out server.crt -keyout server.key \
	-subj "/OU=DaF/CN=RelayerServer"

openssl req -new \
	-newkey rsa:4096 -x509 \
	-sha256 -days 3650 \
	-nodes -out client.crt -keyout client.key \
	-subj "/OU=DaF/CN=RelayerClient"

# Field list: /C=/ST=/L=/O=/OU=/CN=
