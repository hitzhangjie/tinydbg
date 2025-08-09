#!/bin/bash -e


ulimit -c unlimited
sudo sysctl -w kernel.core_pattern=demo.core
LD_LIBRARY_PATH=.:$LD_LIBRARY_PATH GOTRACEBACK=crash ./demo
