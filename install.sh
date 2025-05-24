#!/bin/bash -e

cd cmd/dlv
go build -v -gcflags 'all=-N -l'
cp -f dlv ~/go/bin/tinydbg
