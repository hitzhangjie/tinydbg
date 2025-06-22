```bash
tinydbg/service/rpc2(*RPCServer).Eval(arg EvalIn, out *EvalOut) error
    \--> cfg = &api.LoadConfig{FollowPointers: true, ...)
    |       \--> pcfg := *api.LoadConfigToProc(cfg)
    \--> v, _ := s.debugger.EvalVariableInScope(arg.Scope.GoroutineID, arg.Scope.Frame, arg.Scope.DeferredCall, arg.Expr, pcfg)
    |       \--> s, err := proc.ConvertEvalScope(d.target.Selected, goid, frame, deferredCall)
    |       \--> return s.EvalExpression(expr, cfg)
    |               // 执行完编译之后，ctx.ops将包含3个操作，入栈ident{a}，入栈ident{b}，执行+计算，活脱脱一个后缀表达式
    |               // 但是这里的ctx.ops并不是最终要执行的执行
    |               \--> ops, err := evalop.Compile(scopeToEvalLookup{scope}, expr, scope.evalopFlags())
    |               |       // ast分析得到ast.Expr
    |               |       \--> t, err := parser.ParseExpr(expr)
    |               |       // 对ast.Expr进行编译
    |               |       \--> return CompileAST(lookup, t, flags)
    |               |               \--> err := ctx.compileAST(t, true)
    |               |                       \--> `a+b` operator `+`: case *ast.BinaryExpr: err := ctx.compileBinary(node.X, node.Y, sop, &Binary{node})
    |               |                               \--> operand `a`: err := ctx.compileAST(a, false)
    |               |                                       \--> ctx.pushOp(&PushIdent{node.Name})
    |               |                                           \--> ctx.ops = append(ctx.ops, op)
    |               |                               \--> operand `b`: err := ctx.compileAST(b, false)
    |               |                                       \--> ctx.pushOp(&PushIdent{node.Name})
    |               |                                           \--> ctx.ops = append(ctx.ops, op)
    |               |                                           \--> ctx.ops = append(ctx.ops, op)
    |               |                               \--> operator `+`: ctx.pushOp(op) // `op` is `&Binary{node}`
    |               |               // `ctx.pushOp(op OP)`放到ctx.ops里的每一个操作，都是OP接口的实现，
    |               |               // OP接口要求各个操作汇报各自的popstack、pushstack的次数，
    |               |               // 如：
    |               |               // - pushIdent分别popstack 0次，pushstack 1次，因为仅需要入栈1个参数；
    |               |               // - Binary则是popstack 2次，pushstack 1次，因为二元运算符要通过2次popstack得到2个参数，结果再入栈1次；
    |               |               // 这里的栈深度检查，即校验这些操作执行完后，目标栈深度是否符合预期，如果不符合预期那设计的操作指令有问题。
    |               |               \--> err = ctx.depthCheck(1)
    |               |               \--> return ctx.ops

    |               // 初始化一个栈机器，它讲执行上述ctx.ops里的操作
    |               \--> stack := &evalStack{}
    |               // 栈机器开始执行ctx.ops里的操作
    |               \--> stack.eval(scope, ops)
    |               |        \--> stack.ops = ops
    |               |        \--> stack.scope = scope
    |               |        \--> stack.spoff = ... / stack.bpoff = ... / stack.fboff = ... / stack.curthread = ...
    |               |        \--> stack.run()
    |               |                // About Execute the instructions:
    |               |                // 1. 1st, 2nd op are pushIdent{a} and pushIdent{b}, they're processed in same branch `case *evalop.PushIdent`.
    |               |                // Except the fact:
    |               |                // - variable `a` found 2 instances, 1) block 8~12, line 9. and 2) and main.main 3~13, line 4.
    |               |                // - variable `b` found 1 instance, main.main 3~13, line 5
    |               |                // 2. 3rd op is pushBinary{+}, it's processed in branch case *evalop.Binary
    |               |                \--> for stack.opidx < len(stack.ops) && stack.err == nil foreach op in stack.ops
    |               |                |       \--> stack.executeOp()
    |               |                |               \--> switch op := ops[stack.opidx].(type)
    |               |                |                      case *evalop.PushIdent: 
    |               |                |                      |   found := stack.pushIdent(scope, op.Name)
    |               |                |                      |   \--> find in locals first:
    |               |                |                      |   |    found = stack.pushLocal(scope, name, 0)
    |               |                |                      |   |        \--> vars, err = scope.Locals(0, name)
    |               |                |                      |   |               \--> vars0, err := scope.simpleLocals(flags|rangeBodyFlags, wantedName)
    |               |                |                      |   |                       \--> dwarfTree, err := scope.image().getDwarfTree(scope.Fn.offset)
    |               |                |                      |   |                       \--> varEntries := reader.Variables(dwarfTree, scope.PC, scope.Line, variablesFlags)
    |               |                |                      |   |                               \--> variablesInternal(nil, root, 0, pc, line, flags, true)
    |               |                |                      |   |                               |        \--> search main.main scope, including blocks within main.main scope
    |               |                |                      |   |                               |             case dwarf.TagLexDwarfBlock, dwarf.TagSubprogram:
    |               |                |                      |   |                               |                \--> if (flags&VariablesOnlyVisible == 0) || root.ContainsPC(pc) then
    |               |                |                      |   |                               |                     check each children of root.Children: 
    |               |                |                      |   |                               |                     v = variablesInternal(v, child, depth+1, pc, line, flags, false)
    |               |                |                      |   |                               \--> return varEntries
    |               |                |                      |   |                        \--> vars := make([]*Variable, 0, len(varEntries))vars []
    |               |                |                      |   |                        \--> foreach var in varEntries
    |               |                |                      |   |                               \--> var, err := extractVarInfoFromEntry(scope.target, scope.BinInfo, scope.image(), scope.Regs, scope.Mem, entry.Tree, scope.dictAddr)
    |               |                |                      |   |                                         \--> 由DIE.DW_ATTR_type读取类型信息: n, t, _ := readVarEntry(entry, image)
    |               |                |                      |   |                                         \--> 由DIE.DW_ATTR_location计算地址: addr, _, _, := bi.Location(entry, dwarf.AttrLocation, regs.PC(), regs, mem)
    |               |                |                      |   |                                        \--> v := newVariable(n, uint64(addr), t, bi, mem)
    |               |                |                      |   |                                        \--> return v
    |               |                |                      |   |                               vars = append(vars, var)
    |               |                |                      |   |                       \--> sort.Stable(&variablesByDepthAndDeclLine{vars, depths})
    |               |                |                      |   |                       \--> mark vars `flags|=VariableShadowed` if shadowed
    |               |                |                      |   |                       \--> return vars
    |               |                |                      |   |               \--> only keep the lastseen one in vars0
    |               |                |                      |   |                    that's the one defined in expected scope,
    |               |                |                      |   |               \--> foreach var in vars
    |               |                |                      |   |                       found := varflags&VariableShadowed == 0
    |               |                |                      |   |                        if found then 
    |               |                |                      |   |                           stack.push(vars[i]) 
    |               |                |                      |   |                               \--> stack.stack = append(stack.stack, v)
    |               |                |                      |   |                           break
    |               |                |                      |   |               \--> return found
    |               |                |                      |   \--> if `found` in locals then return
    |               |                |                      |   \--> if `!found` in locals then find in globals
    |               |                |                      |            v, err := scope.findGlobal(scope.Fn.PackageName(), name)
    |               |                |                      |                \--> will search the package variables, if found then returns them
    |               |                |                      |                \--> if not found, then search the package functions, if found then returns them
    |               |                |                      |                \--> if not found, then search the package constants, if found then returns them
    |               |                |                      case *evalop.Binary:
    |               |                |                      |   scope.evalBinary(op, stack)
    |               |                |                      |   // 从操作数栈stack.stack获取运算符的左右操作数
    |               |                |                      |   \--> yv := stack.pop(); 
    |               |                |                      |           \--> v := s.stack[len(s.stack)-1]
    |               |                |                      |           \--> s.stack = s.stack[:len(s.stack)-1]
    |               |                |                      |           \--> return v
    |               |                |                      |   \--> xv := stack.pop()
    |               |                |                      |           \--> the same as yv := stack.pop()
    |               |                |                      |   // 根据Variable中记录的地址信息（DWARF DW_ATTR_location计算而来），
    |               |                |                      |   \--> xv.loadValue(...); 
    |               |                |                      |           \--> v.loadValueInternal(0, cfg)
    |               |                |                      |                   \--> if v.Kind == reflect.Int, ..., reflect.Int64 then
    |               |                |                      |                        var val int64
    |               |                |                      |                        val, v.Unreadable = readIntRaw(v.mem, v.Addr, v.RealType.(*godwarf.IntType).ByteSize)
    |               |                |                      |                           \--> val := make([]byte, int(size))
    |               |                |                      |                           \--> _, err := mem.ReadMemory(val, addr)
    |               |                |                      |                                   \--> n, _ = processVmRead(t.ID, uintptr(addr), data)
    |               |                |                      |                                       uses syscall SYS_PROCESS_VM_READV, maybe failed
    |               |                |                      |                                   \-- if n == 0 then use syscall ptrace(PTRACE_PEEKDATA, ...)
    |               |                |                      |                                       t.dbp.execPtraceFunc(func() { n, err = sys.PtracePeekData(t.ID, uintptr(addr), data) })
    |               |                |                      |                           \--> n = int64(binary.LittleEndian.Uint64(val))
    |               |                |                      |                           \--> return n
    |               |                |                      |                        v.Value = constant.MakeInt64(val)
    |               |                |                      |   \--> yv.loadValue(...)
    |               |                |                      |           \--> the same as xv.loadValue(...)
    |               |                |                      |   // 校验运算符左右操作数类型是否一致
    |               |                |                      |   \--> typ, err := negotiateType(node.Op, xv, yv)
    |               |                |                      |   // 拿到计算结果
    |               |                |                      |   \--> rc, err := constantBinaryOp(op, xv.Value, yv.Value)
    |               |                |                      |           \--> if op isn't token.SHL, token.SHR then
    |               |                |                      |                r = constant.BinaryOp(x, op, y)
    |               |                |                      |                   \--> a := int64(x), b := int64(y)
    |               |                |                      |                   \--> if op == token.Add then c = a+b
    |               |                |                      |                   \--> return int64Val(c)
    |               |                |                      |           \--> return r
    |               |                |                      |   // 结果变量类型应该和操作数相同，构造一个新结果变量
    |               |                |                      |   \--> r := xv.newVariable("", 0, typ, scope.Mem)
    |               |                |                      |   \--> r.Value = rc
    |               |                |                      |   // 确保计算结果符合目标平台的限制, see: convertInt中符号位扩展、截断处理逻辑
    |               |                |                      |   \--> switch r.Kind
    |               |                |                      |           case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
    |               |                |                      |               \--> n, _ := constant.Int64Val(r.Value)
    |               |                |                      |               \--> r.Value = constant.MakeInt64(int64(convertInt(uint64(n), true, typ.Size())))
    |               |                |                      |   // 最后，将计算结果放入操作数栈
    |               |                |                      |   \--> stack.push(r)
    |               |                |                      |           \--> s.stack = append(s.stack, v)
    |               |                \--> check fncalls ... ignored here
    |               // 栈机器的操作数栈栈顶就是最终计算结果，取出这个变量，这个变量是个计算结果，ev.loaded=true，不用读进程内存进行加载
    |               \--> ev, err := stack.result(&cfg)
    |               |   \--> r = stack.peek()
    |               |   \--> r.loadValue(*cfg)
    |               |           \--> v.loadValueInternal(0, cfg)
    |               |                   // r这个结果变量，是调试器进程构造出来的，r的结果不存储在被调试进程中,
    |               |                   // 所以这里 `v.Addr == 0 && v.Base == 0` 成立，无需从被调试进程内存中加载，直接返回 
    |               |                   \--> if v.Unreadable != nil || v.loaded || (v.Addr == 0 && v.Base == 0) then return
    |               |   \--> return r
    |               // 这次loadValue对这里的场景来说，有点多余
    |               \--> ev.loadValue(cfg)
    |           \--> return ev, nil
    // 将Eval的结果proc.Variable转换成客户端可读的信息api.Variable
    \--> out.Variable = api.ConvertVar(v)
    |       \--> r := Variable{Addr, Name, Kind, Len, Cap, ...}
    |       \--> r.Type = PrettyTypeName(v.DwarfType)
    |               \--> godwarf.Type.String(), 这里就是int
    |       \--> r.Value = VariableValueAsString(v), 这里就是a+b的结果字符串"300"
    |       \--> return &r
```

