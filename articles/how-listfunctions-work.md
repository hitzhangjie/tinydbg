# tinydbg中ListFunctions的工作原理

`ListFunctions`是tinydbg中的一个强大功能,它允许用户搜索和列出目标进程中的函数。本文将详细解释其工作原理。

## 1. 请求和响应参数类型

`ListFunctions` RPC调用接受两个参数:

```go
type ListFunctionsIn struct {
    Filter      string  // 用于过滤函数名的正则表达式模式
    FollowCalls int     // 跟踪函数调用的深度(0表示不跟踪)
}

type ListFunctionsOut struct {
    Funcs []string      // 匹配的函数名列表
}
```

## 2. 正则表达式过滤

函数名过滤使用正则表达式实现。当提供过滤模式时,它会被编译成正则表达式对象:

```go
regex, err := regexp.Compile(filter)
if err != nil {
    return nil, fmt.Errorf("invalid filter argument: %s", err.Error())
}
```

这允许用户使用以下模式搜索函数:
- `main.*` - 所有以"main"开头的函数
- `.*Handler` - 所有以"Handler"结尾的函数
- `[A-Z].*` - 所有导出的函数

## 3. 二进制信息读取

函数信息从目标二进制文件的调试信息(DWARF)中读取。这些信息在调试器初始化时加载并存储在`BinaryInfo`结构中。主要组件包括:

- `Functions` 切片,包含二进制文件中的所有函数
- `Sources` 切片,包含所有源文件
- DWARF调试信息,用于详细的函数元数据

## 4. 函数信息提取

函数信息在调试器初始化期间从DWARF调试信息中提取。对于每个函数,存储以下信息:

```go
type Function struct {
    Name       string
    Entry, End uint64    // 函数地址范围
    offset     dwarf.Offset
    cu         *compileUnit
    trampoline bool
    InlinedCalls []InlinedCall
}
```

## 5. 函数调用遍历

当`FollowCalls`大于0时,调试器会执行函数调用的广度优先遍历。这是在`traverse`函数中实现的:

```go
func traverse(t proc.ValidTargets, f *proc.Function, depth int, followCalls int) ([]string, error) {
    type TraceFunc struct {
        Func    *proc.Function
        Depth   int
        visited bool
    }
    
    // 使用map跟踪已访问的函数,避免循环
    TraceMap := make(map[string]TraceFuncptr)
    queue := make([]TraceFuncptr, 0, 40)
    funcs := []string{}
    
    // 从根函数开始
    rootnode := &TraceFunc{Func: f, Depth: depth, visited: false}
    TraceMap[f.Name] = rootnode
    queue = append(queue, rootnode)
    
    // BFS遍历
    for len(queue) > 0 {
        parent := queue[0]
        queue = queue[1:]
        
        // 如果超过调用深度则跳过
        if parent.Depth > followCalls {
            continue
        }
        
        // 如果已访问则跳过
        if parent.visited {
            continue
        }
        
        funcs = append(funcs, parent.Func.Name)
        parent.visited = true
        
        // 反汇编函数以查找调用
        text, err := proc.Disassemble(t.Memory(), nil, t.Breakpoints(), t.BinInfo(), f.Entry, f.End)
        if err != nil {
            return nil, err
        }
        
        // 处理每条指令
        for _, instr := range text {
            if instr.IsCall() && instr.DestLoc != nil && instr.DestLoc.Fn != nil {
                cf := instr.DestLoc.Fn
                // 跳过大多数runtime函数,除了特定的几个
                if (strings.HasPrefix(cf.Name, "runtime.") || strings.HasPrefix(cf.Name, "runtime/internal")) &&
                    cf.Name != "runtime.deferreturn" && cf.Name != "runtime.gorecover" && cf.Name != "runtime.gopanic" {
                    continue
                }
                
                // 如果未访问过,将新函数添加到队列
                if TraceMap[cf.Name] == nil {
                    childnode := &TraceFunc{Func: cf, Depth: parent.Depth + 1, visited: false}
                    TraceMap[cf.Name] = childnode
                    queue = append(queue, childnode)
                }
            }
        }
    }
    return funcs, nil
}
```

遍历算法:
1. 使用map跟踪已访问的函数，避免重复访问
2. 使用队列进行广度优先遍历
3. 对于每个函数:
   - 反汇编其代码
   - 查找所有CALL指令
   - 提取被调用函数的信息
   - 如果未访问过,将新函数添加到队列
4. 跳过大多数runtime函数以减少干扰
5. 遵守最大调用深度参数

ps: 这里为什么不使用AST呢？查找FuncDecl.Body中的所有函数调用，不也是一种办法，确实也是一种办法。但是通过AST的方式应该效率会很慢，而且由于存在内联，AST中的结构不一定能反映最终编译优化后的指令，比如内联优化。使用AST当我们尝试对某个函数位置进行trace并获取这个函数参数时，可能会出现错误，因为它被内联了，通过BP寄存器+参数偏移量的方式获取的不是真实参数。这里使用CALL指令可以避免上述考虑不周的错误，而且处理效率会更高效。

## 6. 结果处理

最后一步处理结果:

```go
// 排序并删除重复项
sort.Strings(funcs)
funcs = slices.Compact(funcs)
```

这确保返回的函数列表:
- 按字母顺序排序
- 没有重复项
- 只包含匹配过滤模式的函数

## 7. 在调试器命令中的使用

`ListFunctions`功能主要用于两个调试器命令:

1. `funcs <regexp>` - 列出所有匹配模式的函数
2. `trace <regexp>` - 在匹配的函数及其被调用函数上设置跟踪点

例如:
```
tinydbg> funcs main.*
main.main
main.init
main.handleRequest

tinydbg> trace main.*
```

trace命令使用`ListFunctions`并将`FollowCalls`设置为大于0,以查找可能被匹配函数调用的所有函数,从而实现全面的函数调用跟踪。 