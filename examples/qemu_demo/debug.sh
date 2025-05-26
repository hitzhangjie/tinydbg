#!/bin/bash

# 确保脚本在错误时退出
set -e

# 检查QEMU是否安装
if ! command -v qemu-system-x86_64 &> /dev/null; then
    echo "Error: qemu-system-x86_64 not found. Please install QEMU first."
    echo "On macOS, you can install it using: brew install qemu"
    exit 1
fi

# 检查GDB是否安装
if ! command -v gdb &> /dev/null; then
    echo "Error: gdb not found. Please install GDB first."
    echo "On macOS, you can install it using: brew install gdb"
    exit 1
fi

# 启动QEMU的GDB服务器
echo "Starting QEMU GDB server..."
qemu-system-x86_64 -kernel ./test_program -S -gdb tcp::1234 &

# 等待QEMU启动
sleep 1

# 启动GDB并连接到QEMU
echo "Starting GDB..."
gdb -ex "set architecture i386:x86-64" \
    -ex "target remote localhost:1234" \
    -ex "file ./test_program" \
    -ex "break main" \
    -ex "continue" \
    -ex "layout src" \
    ./test_program

# 清理：当GDB退出时，终止QEMU进程
pkill -f "qemu-system-x86_64.*test_program" 
