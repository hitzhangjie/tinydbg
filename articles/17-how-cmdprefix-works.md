# TinyDBG 中的 cmdPrefix 机制

## 简介

在 TinyDBG 中，`cmdPrefix` 是一个允许命令在不同上下文中执行的机制。本文将解释 `cmdPrefix` 的工作原理、设计目的以及在命令执行中的作用。

## 什么是 cmdPrefix？

`cmdPrefix` 是一个整数类型，用于定义不同的命令执行上下文：

```go
type cmdPrefix int

const (
    noPrefix = cmdPrefix(0)        // 无前缀
    onPrefix = cmdPrefix(1 << iota) // 开启前缀
    deferredPrefix                  // 延迟前缀
    revPrefix                      // 反向前缀
)
```

## 命令结构

TinyDBG 中的每个命令都有一个 `allowedPrefixes` 字段，用于指定该命令支持哪些前缀：

```go
type command struct {
    aliases         []string
    builtinAliases  []string
    group           commandGroup
    allowedPrefixes cmdPrefix  // 支持的前缀
    helpMsg         string
    cmdFn           cmdfunc
}
```

## 前缀类型及其用途

### 1. noPrefix (0)
- 当不需要特定上下文时使用的默认前缀
- 大多数基本命令使用此外缀
- 示例：用于打印包变量的 `vars` 命令

### 2. onPrefix (1 << iota)
- 在断点处执行命令时使用
- 允许命令访问断点上下文
- 示例：`print` 命令可以访问断点位置的变量
- 示例：`stack` 命令可以显示断点位置的堆栈跟踪

### 3. deferredPrefix
- 在延迟函数上下文中执行命令时使用
- 允许检查延迟函数的执行
- 示例：`print` 和 `args` 命令可以显示延迟函数中的变量

### 4. revPrefix
- 最初设计用于支持反向调试
- 由于反向调试功能已被移除，目前不再使用
- 原本用于 `continue`、`step` 和 `next` 等命令以支持向后执行
- 可能在未来的版本中被移除

## 命令执行上下文

命令在包含前缀的 `callContext` 中执行：

```go
type callContext struct {
    Prefix     cmdPrefix
    Scope      api.EvalScope
    Breakpoint *api.Breakpoint
}
```

## 前缀检查

执行命令时，TinyDBG 会检查命令是否支持当前前缀：

```go
func (s *DebugSession) Find(cmdstr string, prefix cmdPrefix) *command {
    for _, v := range s.cmds {
        if v.match(cmdstr) {
            if prefix != noPrefix && v.allowedPrefixes&prefix == 0 {
                continue
            }
            return v
        }
    }
    return &command{aliases: []string{"nocmd"}, cmdFn: noCmdAvailable}
}
```

## 命令示例及其前缀

### 数据检查命令
- `print`：支持 `onPrefix | deferredPrefix`
- `args`：支持 `onPrefix | deferredPrefix`
- `locals`：支持 `onPrefix | deferredPrefix`
- `vars`：不支持前缀（使用 `noPrefix`）

### 执行控制命令
- `continue`：当前支持 `revPrefix`（可能被移除）
- `step`：当前支持 `revPrefix`（可能被移除）
- `next`：当前支持 `revPrefix`（可能被移除）

### 堆栈命令
- `stack`：支持 `onPrefix`
- `frame`：不支持前缀（使用 `noPrefix`）

## 设计目的

`cmdPrefix` 机制服务于以下几个重要目的：

1. **上下文感知**：允许命令了解其执行上下文（断点、延迟函数等）

2. **命令复用**：使同一个命令能够根据上下文表现出不同的行为

3. **安全性**：防止命令在不适当的上下文中执行

4. **可扩展性**：便于在未来添加新的执行上下文

## 未来考虑

1. 由于反向调试支持已被移除，`revPrefix` 可能被移除

2. 如果需要，可以添加新的前缀来支持其他执行上下文

3. 前缀系统可以扩展以支持更复杂的上下文组合

## 结论

TinyDBG 中的 `cmdPrefix` 系统提供了一种灵活而强大的方式来处理不同的命令执行上下文。虽然反向调试等功能已被移除，但核心的前缀机制仍然是调试器架构的重要组成部分。 