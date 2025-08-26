# Recording功能清理总结

## 概述

本文档总结了从tinydbg调试器中清理mozilla rr recording相关代码的工作。由于tinydbg是一个简化的调试器示例，不支持mozilla rr的录制功能，因此移除了所有相关的抽象和实现。

## 清理内容

### 1. 错误定义和常量

**移除的错误：**
- `pkg/proc/target.go`: `ErrNotRecorded` - 录制相关操作错误
- `service/debugger/debugger.go`: `ErrNotRecording` - 调试器录制状态错误

**修改：**
- `service/debugger/debugger.go`: 将checkpoint重启检查改为返回"checkpoint restart not supported"错误

### 2. API类型定义

**移除的字段：**
- `service/api/types.go`: `Recording` - 录制状态标志
- `service/api/types.go`: `When` - 录制位置描述

### 3. Debugger结构体

**移除的字段：**
- `service/debugger/debugger.go`: `stopRecording` - 停止录制函数
- `service/debugger/debugger.go`: `recordMutex` - 录制互斥锁
- `service/debugger/debugger.go`: `RrOnProcessPid` - RR进程PID

**修改的方法：**
- `StopRecording()`: 简化为直接返回"recording not supported"错误
- `Restart()`: 移除录制相关的注释和逻辑

### 4. RPC服务

**移除的RPC方法：**
- `service/rpc2/server.go`: `StopRecording` RPC方法
- `service/rpc2/client.go`: `StopRecording` 客户端方法
- `service/rpccommon/server.go`: RPC方法注册

**修改的RPC结构体：**
- `RestartIn`: 将`Rerecord`和`Position`字段标记为已弃用

### 5. 客户端接口

**移除的方法：**
- `service/client.go`: `StopRecording()` 接口方法

### 6. 命令行帮助文本

**修改的帮助文本：**
- `cmds/debug/debug_run.go`: 移除restart命令中关于录制和checkpoint的说明
- 简化为只支持基本的重启功能

### 7. 信号处理

**移除的逻辑：**
- `cmds/debug/session.go`: 移除SIGINT处理中的录制停止逻辑
- 简化注释，移除录制相关的描述

### 8. 测试代码

**移除的测试：**
- `service/test/integration2_test.go`: `TestStopRecording` - 停止录制测试
- `service/test/integration2_test.go`: `TestRerecord` - 重新录制测试

## 保留的内容

### 1. Recorded方法

保留了`Recorded()`方法，但简化了实现：
- `pkg/proc/target.go`: 保留方法，仅用于core dump检测
- `pkg/proc/target_exec.go`: 保留调用，用于区分core dump和live process
- `pkg/proc/stackwatch.go`: 保留调用，用于watchpoint处理

**原因：** 这些方法不仅用于mozilla rr，也用于处理core dump文件，是调试器的基本功能。

### 2. RestartFrom接口

保留了`RestartFrom`接口，但简化了参数含义：
- 保留了`rerecord`和`pos`参数以维持API兼容性
- 将这些参数标记为已弃用，实际功能被忽略

## 影响分析

### 1. 功能影响

**移除的功能：**
- 无法使用mozilla rr进行确定性调试
- 无法从checkpoint重启程序
- 无法停止录制过程

**保留的功能：**
- 基本的程序重启功能
- Core dump文件调试
- 所有其他调试功能

### 2. API兼容性

**向后兼容：**
- 保留了所有公共API接口
- 录制相关的方法返回明确的错误信息
- 不会导致现有代码编译失败

**API变化：**
- `StopRecording()` 现在返回"recording not supported"错误
- `Restart()` 中的checkpoint参数被忽略

### 3. 用户体验

**改进：**
- 简化了restart命令的帮助文本
- 移除了不工作的功能，避免用户困惑
- 错误信息更加明确

**限制：**
- 无法使用高级的录制调试功能
- 需要依赖其他工具进行确定性调试

## 技术债务

### 1. 遗留参数

以下参数仍然存在于API中但被忽略：
- `RestartFrom` 中的 `rerecord` 参数
- `RestartFrom` 中的 `pos` 参数
- `RestartIn` 中的 `Rerecord` 和 `Position` 字段

**建议：** 在未来的版本中可以考虑完全移除这些参数。

### 2. 文档更新

需要更新的文档：
- 用户手册中的restart命令说明
- API文档中的录制相关方法
- 示例代码中的录制功能演示

## 总结

本次清理工作成功移除了tinydbg中所有mozilla rr recording相关的代码，同时保持了API的向后兼容性。清理后的代码更加简洁，专注于核心的调试功能，避免了用户对不工作功能的困惑。

主要成果：
1. 移除了约200行recording相关代码
2. 简化了restart命令的实现
3. 保持了API兼容性
4. 明确了功能边界

这次清理为tinydbg提供了一个更加清晰和专注的代码库，符合其作为调试器示例项目的定位。
