# Starlark 调试演示

这个演示展示了如何使用 Starlark 脚本来扩展调试器的功能。

## 文件说明

- `main.go`: 一个简单的 Go 程序，作为调试目标
- `debug.star`: Starlark 调试脚本，展示了如何使用 Starlark 进行调试

## 如何使用

1. 首先编译示例程序：
   ```bash
   go build main.go
   ```

2. 使用调试器启动程序：
   ```bash
   tinydbg ./main
   ```

3. 在调试器中使用 Starlark 脚本：
   ```bash
   (tinydbg) source debug.star
   ```

## 脚本功能说明

这个 Starlark 脚本展示了以下功能：

1. 使用 `dlv_command` 执行调试器命令
2. 使用 `cur_scope` 获取当前作用域信息
3. 使用 `write_file` 保存调试信息
4. 定义和调用自定义函数
5. 自动化调试流程

## 脚本执行流程

1. 在 `main` 函数处设置断点
2. 继续执行程序
3. 在 `processPerson` 函数处设置断点
4. 打印变量值和作用域信息
5. 保存调试信息到文件

## 注意事项

- 确保调试器支持 Starlark 脚本
- 脚本中的命令会根据程序的实际执行情况而有所不同
- 可以根据需要修改脚本来自定义调试行为 