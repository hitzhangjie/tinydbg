#!/bin/bash

echo "CLOC STAT 1: including *.go + *_test"
cloc cmds/ pkg/ service/
echo ""
echo ""

echo "CLOC STAT 2: including *.go"
cloc --exclude-content=testing.T cmds/ pkg/ service/
