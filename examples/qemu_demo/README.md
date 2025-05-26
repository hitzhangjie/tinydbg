# QEMU GDB 调试测试环境

这个目录包含了使用 QEMU 和 GDB 在 macOS 上调试 Linux x86_64 程序的测试环境。

## 前置要求

1. 安装 QEMU：
```bash
brew install qemu
```

2. 安装 GDB：
```bash
brew install gdb
```

3. 安装交叉编译工具链：
```bash
brew install x86_64-elf-gcc
```

## 使用方法

1. 编译测试程序：
```bash
chmod +x build.sh
./build.sh
```

2. 启动调试会话：
```bash
chmod +x debug.sh
./debug.sh
```

## 调试说明

- 程序会在 `main` 函数处设置断点
- 使用标准的 GDB 命令进行调试：
  - `n` (next): 单步执行
  - `s` (step): 步入函数
  - `c` (continue): 继续执行
  - `p variable`: 打印变量值
  - `bt`: 显示调用栈
  - `info registers`: 显示寄存器值

## 注意事项

1. 确保所有脚本都有执行权限
2. 如果遇到权限问题，可能需要以 root 权限运行 GDB
3. 调试会话结束后，脚本会自动清理 QEMU 进程 