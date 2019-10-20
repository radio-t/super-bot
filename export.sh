#!/bin/sh
echo "export log for $1"
docker exec -i gitter-bot /srv/gitter-rt-bot --super=Umputun --super=bobuk --super=ksenks --super=grayru --dbg --export-num=$1 --export-path=/srv/html
