# 3. 分离式架构下的前后端通信设计与实现

## 3.1 通信模式概述

tinydbg 采用前后端分离架构，前端（UI/CLI/IDE插件等）与后端（核心调试引擎）通过标准协议进行通信。根据实际场景，支持两种主要通信模式：

- **本地模式**：前后端运行于同一进程或主机，通过高效的内存管道通信。
- **远程模式**：前后端分布于不同主机，通过网络协议进行通信，支持跨主机、跨平台调试。

## 3.2 远程调试：基于json-rpc的设计与实现

在远程调试场景下，tinydbg 前后端通过 json-rpc 协议进行通信。其核心设计如下：

- **协议选择**：json-rpc 是一种轻量级、跨语言的远程过程调用协议，易于解析和扩展，适合调试器前后端解耦。
- **消息结构**：所有调试命令、事件、响应均以 JSON 格式封装，通过 TCP/Unix Socket 等网络通道传输。
- **典型流程**：
  1. 前端发起调试命令（如设置断点、单步执行），序列化为 json-rpc 请求。
  2. 后端接收请求，解析并执行业务逻辑，返回结果或事件通知。
  3. 前端解析响应，更新UI或输出结果。
- **关键实现片段**：

```go
// 伪代码示例：json-rpc服务端注册与处理
rpcServer := jsonrpc.NewServer()
rpcServer.Register("SetBreakpoint", setBreakpointHandler)
rpcServer.Register("Continue", continueHandler)
// ...
listener, _ := net.Listen("tcp", ":12345")
rpcServer.Serve(listener)
```

- **优势**：
  - 支持多种前端（CLI、Web、IDE等）灵活对接
  - 易于扩展新命令和事件
  - 支持跨主机、跨平台调试

## 3.3 本地调试：preconnectedPipe（net.Pipe）机制

在本地调试场景下，为了避免网络通信带来的额外开销，tinydbg 采用 Go 标准库的 `net.Pipe` 实现前后端的高效内存通道：

- **机制简介**：`net.Pipe` 提供一对内存中的全双工连接，模拟网络 socket，但无实际网络传输延迟。
- **典型流程**：
  1. 启动调试器时，前后端通过 `net.Pipe` 创建一对连接端。
  2. 前端与后端分别持有一端，直接通过内存通道进行消息交换。
  3. 保持与远程模式一致的 json-rpc 消息格式，便于代码复用。
- **关键实现片段**：

```go
// 伪代码示例：本地pipe创建
frontendConn, backendConn := net.Pipe()
go frontend.Run(frontendConn)
go backend.Run(backendConn)
```

- **优势**：
  - 零拷贝、低延迟，极大提升本地调试性能
  - 代码与远程模式高度复用，易于维护

## 3.4 通信安全与健壮性

- 远程模式下可集成认证、加密等安全机制，保障生产环境安全。
- 支持心跳检测、异常断线重连等机制，提升健壮性。

通过上述设计，tinydbg 实现了灵活、高效、可扩展的前后端通信，为多场景调试需求提供了坚实基础。 