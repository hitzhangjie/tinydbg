#!/bin/bash

# 确保脚本在错误时退出
set -e

echo "Building test program for Linux x86_64..."

# 使用交叉编译工具链编译
x86_64-linux-gnu-gcc -g -static -o test_program test_program.c

echo "Build complete. Binary created: test_program" 
