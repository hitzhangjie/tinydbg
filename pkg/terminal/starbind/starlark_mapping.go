// DO NOT EDIT: auto-generated using _scripts/gen-starlark-bindings.go

package starbind

import (
	"fmt"
	"github.com/hitzhangjie/dlv/service"
	"github.com/hitzhangjie/dlv/service/api"
	"go.starlark.net/starlark"
)

func (env *Env) starlarkPredeclare() starlark.StringDict {
	r := starlark.StringDict{}

	r["amend_breakpoint"] = starlark.NewBuiltin("amend_breakpoint", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.AmendBreakpointIn
		var rpcRet service.AmendBreakpointOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Breakpoint, "Breakpoint")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Breakpoint":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Breakpoint, "Breakpoint")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("AmendBreakpoint", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["ancestors"] = starlark.NewBuiltin("ancestors", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.AncestorsIn
		var rpcRet service.AncestorsOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.GoroutineID, "GoroutineID")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 1 && args[1] != starlark.None {
			err := unmarshalStarlarkValue(args[1], &rpcArgs.NumAncestors, "NumAncestors")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 2 && args[2] != starlark.None {
			err := unmarshalStarlarkValue(args[2], &rpcArgs.Depth, "Depth")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "GoroutineID":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.GoroutineID, "GoroutineID")
			case "NumAncestors":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.NumAncestors, "NumAncestors")
			case "Depth":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Depth, "Depth")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("Ancestors", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["attached_to_existing_process"] = starlark.NewBuiltin("attached_to_existing_process", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.AttachedToExistingProcessIn
		var rpcRet service.AttachedToExistingProcessOut
		err := env.ctx.Client().CallAPI("AttachedToExistingProcess", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["cancel_next"] = starlark.NewBuiltin("cancel_next", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.CancelNextIn
		var rpcRet service.CancelNextOut
		err := env.ctx.Client().CallAPI("CancelNext", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["clear_breakpoint"] = starlark.NewBuiltin("clear_breakpoint", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.ClearBreakpointIn
		var rpcRet service.ClearBreakpointOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Id, "Id")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 1 && args[1] != starlark.None {
			err := unmarshalStarlarkValue(args[1], &rpcArgs.Name, "Name")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Id":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Id, "Id")
			case "Name":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Name, "Name")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("ClearBreakpoint", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["raw_command"] = starlark.NewBuiltin("raw_command", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs api.DebuggerCommand
		var rpcRet service.CommandOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Name, "Name")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 1 && args[1] != starlark.None {
			err := unmarshalStarlarkValue(args[1], &rpcArgs.ThreadID, "ThreadID")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 2 && args[2] != starlark.None {
			err := unmarshalStarlarkValue(args[2], &rpcArgs.GoroutineID, "GoroutineID")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 3 && args[3] != starlark.None {
			err := unmarshalStarlarkValue(args[3], &rpcArgs.ReturnInfoLoadConfig, "ReturnInfoLoadConfig")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		} else {
			cfg := env.ctx.LoadConfig()
			rpcArgs.ReturnInfoLoadConfig = &cfg
		}
		if len(args) > 4 && args[4] != starlark.None {
			err := unmarshalStarlarkValue(args[4], &rpcArgs.Expr, "Expr")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 5 && args[5] != starlark.None {
			err := unmarshalStarlarkValue(args[5], &rpcArgs.UnsafeCall, "UnsafeCall")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Name":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Name, "Name")
			case "ThreadID":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.ThreadID, "ThreadID")
			case "GoroutineID":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.GoroutineID, "GoroutineID")
			case "ReturnInfoLoadConfig":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.ReturnInfoLoadConfig, "ReturnInfoLoadConfig")
			case "Expr":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Expr, "Expr")
			case "UnsafeCall":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.UnsafeCall, "UnsafeCall")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("Command", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["create_breakpoint"] = starlark.NewBuiltin("create_breakpoint", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.CreateBreakpointIn
		var rpcRet service.CreateBreakpointOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Breakpoint, "Breakpoint")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Breakpoint":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Breakpoint, "Breakpoint")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("CreateBreakpoint", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["create_ebpf_tracepoint"] = starlark.NewBuiltin("create_ebpf_tracepoint", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.CreateEBPFTracepointIn
		var rpcRet service.CreateEBPFTracepointOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.FunctionName, "FunctionName")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "FunctionName":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.FunctionName, "FunctionName")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("CreateEBPFTracepoint", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["create_watchpoint"] = starlark.NewBuiltin("create_watchpoint", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.CreateWatchpointIn
		var rpcRet service.CreateWatchpointOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Scope, "Scope")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		} else {
			rpcArgs.Scope = env.ctx.Scope()
		}
		if len(args) > 1 && args[1] != starlark.None {
			err := unmarshalStarlarkValue(args[1], &rpcArgs.Expr, "Expr")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 2 && args[2] != starlark.None {
			err := unmarshalStarlarkValue(args[2], &rpcArgs.Type, "Type")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Scope":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Scope, "Scope")
			case "Expr":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Expr, "Expr")
			case "Type":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Type, "Type")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("CreateWatchpoint", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["detach"] = starlark.NewBuiltin("detach", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.DetachIn
		var rpcRet service.DetachOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Kill, "Kill")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Kill":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Kill, "Kill")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("Detach", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["disassemble"] = starlark.NewBuiltin("disassemble", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.DisassembleIn
		var rpcRet service.DisassembleOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Scope, "Scope")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		} else {
			rpcArgs.Scope = env.ctx.Scope()
		}
		if len(args) > 1 && args[1] != starlark.None {
			err := unmarshalStarlarkValue(args[1], &rpcArgs.StartPC, "StartPC")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 2 && args[2] != starlark.None {
			err := unmarshalStarlarkValue(args[2], &rpcArgs.EndPC, "EndPC")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 3 && args[3] != starlark.None {
			err := unmarshalStarlarkValue(args[3], &rpcArgs.Flavour, "Flavour")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Scope":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Scope, "Scope")
			case "StartPC":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.StartPC, "StartPC")
			case "EndPC":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.EndPC, "EndPC")
			case "Flavour":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Flavour, "Flavour")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("Disassemble", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["dump_cancel"] = starlark.NewBuiltin("dump_cancel", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.DumpCancelIn
		var rpcRet service.DumpCancelOut
		err := env.ctx.Client().CallAPI("DumpCancel", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["dump_start"] = starlark.NewBuiltin("dump_start", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.DumpStartIn
		var rpcRet service.DumpStartOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Destination, "Destination")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Destination":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Destination, "Destination")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("DumpStart", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["dump_wait"] = starlark.NewBuiltin("dump_wait", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.DumpWaitIn
		var rpcRet service.DumpWaitOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Wait, "Wait")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Wait":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Wait, "Wait")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("DumpWait", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["eval"] = starlark.NewBuiltin("eval", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.EvalIn
		var rpcRet service.EvalOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Scope, "Scope")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		} else {
			rpcArgs.Scope = env.ctx.Scope()
		}
		if len(args) > 1 && args[1] != starlark.None {
			err := unmarshalStarlarkValue(args[1], &rpcArgs.Expr, "Expr")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 2 && args[2] != starlark.None {
			err := unmarshalStarlarkValue(args[2], &rpcArgs.Cfg, "Cfg")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		} else {
			cfg := env.ctx.LoadConfig()
			rpcArgs.Cfg = &cfg
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Scope":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Scope, "Scope")
			case "Expr":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Expr, "Expr")
			case "Cfg":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Cfg, "Cfg")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("Eval", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["examine_memory"] = starlark.NewBuiltin("examine_memory", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.ExamineMemoryIn
		var rpcRet service.ExaminedMemoryOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Address, "Address")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 1 && args[1] != starlark.None {
			err := unmarshalStarlarkValue(args[1], &rpcArgs.Length, "Length")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Address":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Address, "Address")
			case "Length":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Length, "Length")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("ExamineMemory", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["find_location"] = starlark.NewBuiltin("find_location", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.FindLocationIn
		var rpcRet service.FindLocationOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Scope, "Scope")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		} else {
			rpcArgs.Scope = env.ctx.Scope()
		}
		if len(args) > 1 && args[1] != starlark.None {
			err := unmarshalStarlarkValue(args[1], &rpcArgs.Loc, "Loc")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 2 && args[2] != starlark.None {
			err := unmarshalStarlarkValue(args[2], &rpcArgs.IncludeNonExecutableLines, "IncludeNonExecutableLines")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 3 && args[3] != starlark.None {
			err := unmarshalStarlarkValue(args[3], &rpcArgs.SubstitutePathRules, "SubstitutePathRules")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Scope":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Scope, "Scope")
			case "Loc":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Loc, "Loc")
			case "IncludeNonExecutableLines":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.IncludeNonExecutableLines, "IncludeNonExecutableLines")
			case "SubstitutePathRules":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.SubstitutePathRules, "SubstitutePathRules")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("FindLocation", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["function_return_locations"] = starlark.NewBuiltin("function_return_locations", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.FunctionReturnLocationsIn
		var rpcRet service.FunctionReturnLocationsOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.FnName, "FnName")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "FnName":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.FnName, "FnName")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("FunctionReturnLocations", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["get_breakpoint"] = starlark.NewBuiltin("get_breakpoint", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.GetBreakpointIn
		var rpcRet service.GetBreakpointOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Id, "Id")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 1 && args[1] != starlark.None {
			err := unmarshalStarlarkValue(args[1], &rpcArgs.Name, "Name")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Id":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Id, "Id")
			case "Name":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Name, "Name")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("GetBreakpoint", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["get_buffered_tracepoints"] = starlark.NewBuiltin("get_buffered_tracepoints", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.GetBufferedTracepointsIn
		var rpcRet service.GetBufferedTracepointsOut
		err := env.ctx.Client().CallAPI("GetBufferedTracepoints", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["get_thread"] = starlark.NewBuiltin("get_thread", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.GetThreadIn
		var rpcRet service.GetThreadOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Id, "Id")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Id":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Id, "Id")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("GetThread", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["is_multiclient"] = starlark.NewBuiltin("is_multiclient", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.IsMulticlientIn
		var rpcRet service.IsMulticlientOut
		err := env.ctx.Client().CallAPI("IsMulticlient", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["last_modified"] = starlark.NewBuiltin("last_modified", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.LastModifiedIn
		var rpcRet service.LastModifiedOut
		err := env.ctx.Client().CallAPI("LastModified", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["breakpoints"] = starlark.NewBuiltin("breakpoints", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.ListBreakpointsIn
		var rpcRet service.ListBreakpointsOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.All, "All")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "All":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.All, "All")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("ListBreakpoints", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["dynamic_libraries"] = starlark.NewBuiltin("dynamic_libraries", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.ListDynamicLibrariesIn
		var rpcRet service.ListDynamicLibrariesOut
		err := env.ctx.Client().CallAPI("ListDynamicLibraries", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["function_args"] = starlark.NewBuiltin("function_args", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.ListFunctionArgsIn
		var rpcRet service.ListFunctionArgsOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Scope, "Scope")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		} else {
			rpcArgs.Scope = env.ctx.Scope()
		}
		if len(args) > 1 && args[1] != starlark.None {
			err := unmarshalStarlarkValue(args[1], &rpcArgs.Cfg, "Cfg")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		} else {
			rpcArgs.Cfg = env.ctx.LoadConfig()
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Scope":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Scope, "Scope")
			case "Cfg":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Cfg, "Cfg")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("ListFunctionArgs", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["functions"] = starlark.NewBuiltin("functions", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.ListFunctionsIn
		var rpcRet service.ListFunctionsOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Filter, "Filter")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Filter":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Filter, "Filter")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("ListFunctions", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["goroutines"] = starlark.NewBuiltin("goroutines", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.ListGoroutinesIn
		var rpcRet service.ListGoroutinesOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Start, "Start")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 1 && args[1] != starlark.None {
			err := unmarshalStarlarkValue(args[1], &rpcArgs.Count, "Count")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 2 && args[2] != starlark.None {
			err := unmarshalStarlarkValue(args[2], &rpcArgs.Filters, "Filters")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 3 && args[3] != starlark.None {
			err := unmarshalStarlarkValue(args[3], &rpcArgs.GoroutineGroupingOptions, "GoroutineGroupingOptions")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Start":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Start, "Start")
			case "Count":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Count, "Count")
			case "Filters":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Filters, "Filters")
			case "GoroutineGroupingOptions":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.GoroutineGroupingOptions, "GoroutineGroupingOptions")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("ListGoroutines", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["local_vars"] = starlark.NewBuiltin("local_vars", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.ListLocalVarsIn
		var rpcRet service.ListLocalVarsOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Scope, "Scope")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		} else {
			rpcArgs.Scope = env.ctx.Scope()
		}
		if len(args) > 1 && args[1] != starlark.None {
			err := unmarshalStarlarkValue(args[1], &rpcArgs.Cfg, "Cfg")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		} else {
			rpcArgs.Cfg = env.ctx.LoadConfig()
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Scope":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Scope, "Scope")
			case "Cfg":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Cfg, "Cfg")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("ListLocalVars", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["package_vars"] = starlark.NewBuiltin("package_vars", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.ListPackageVarsIn
		var rpcRet service.ListPackageVarsOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Filter, "Filter")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 1 && args[1] != starlark.None {
			err := unmarshalStarlarkValue(args[1], &rpcArgs.Cfg, "Cfg")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		} else {
			rpcArgs.Cfg = env.ctx.LoadConfig()
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Filter":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Filter, "Filter")
			case "Cfg":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Cfg, "Cfg")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("ListPackageVars", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["packages_build_info"] = starlark.NewBuiltin("packages_build_info", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.ListPackagesBuildInfoIn
		var rpcRet service.ListPackagesBuildInfoOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.IncludeFiles, "IncludeFiles")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "IncludeFiles":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.IncludeFiles, "IncludeFiles")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("ListPackagesBuildInfo", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["registers"] = starlark.NewBuiltin("registers", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.ListRegistersIn
		var rpcRet service.ListRegistersOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.ThreadID, "ThreadID")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 1 && args[1] != starlark.None {
			err := unmarshalStarlarkValue(args[1], &rpcArgs.IncludeFp, "IncludeFp")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 2 && args[2] != starlark.None {
			err := unmarshalStarlarkValue(args[2], &rpcArgs.Scope, "Scope")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		} else {
			scope := env.ctx.Scope()
			rpcArgs.Scope = &scope
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "ThreadID":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.ThreadID, "ThreadID")
			case "IncludeFp":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.IncludeFp, "IncludeFp")
			case "Scope":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Scope, "Scope")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("ListRegisters", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["sources"] = starlark.NewBuiltin("sources", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.ListSourcesIn
		var rpcRet service.ListSourcesOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Filter, "Filter")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Filter":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Filter, "Filter")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("ListSources", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["threads"] = starlark.NewBuiltin("threads", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.ListThreadsIn
		var rpcRet service.ListThreadsOut
		err := env.ctx.Client().CallAPI("ListThreads", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["types"] = starlark.NewBuiltin("types", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.ListTypesIn
		var rpcRet service.ListTypesOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Filter, "Filter")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Filter":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Filter, "Filter")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("ListTypes", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["process_pid"] = starlark.NewBuiltin("process_pid", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.ProcessPidIn
		var rpcRet service.ProcessPidOut
		err := env.ctx.Client().CallAPI("ProcessPid", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["restart"] = starlark.NewBuiltin("restart", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.RestartIn
		var rpcRet service.RestartOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Rebuild, "Rebuild")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Rebuild":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Rebuild, "Rebuild")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("Restart", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["set_expr"] = starlark.NewBuiltin("set_expr", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.SetIn
		var rpcRet service.SetOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Scope, "Scope")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		} else {
			rpcArgs.Scope = env.ctx.Scope()
		}
		if len(args) > 1 && args[1] != starlark.None {
			err := unmarshalStarlarkValue(args[1], &rpcArgs.Symbol, "Symbol")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 2 && args[2] != starlark.None {
			err := unmarshalStarlarkValue(args[2], &rpcArgs.Value, "Value")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Scope":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Scope, "Scope")
			case "Symbol":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Symbol, "Symbol")
			case "Value":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Value, "Value")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("Set", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["stacktrace"] = starlark.NewBuiltin("stacktrace", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.StacktraceIn
		var rpcRet service.StacktraceOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Id, "Id")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 1 && args[1] != starlark.None {
			err := unmarshalStarlarkValue(args[1], &rpcArgs.Depth, "Depth")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 2 && args[2] != starlark.None {
			err := unmarshalStarlarkValue(args[2], &rpcArgs.Full, "Full")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 3 && args[3] != starlark.None {
			err := unmarshalStarlarkValue(args[3], &rpcArgs.Defers, "Defers")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 4 && args[4] != starlark.None {
			err := unmarshalStarlarkValue(args[4], &rpcArgs.Opts, "Opts")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 5 && args[5] != starlark.None {
			err := unmarshalStarlarkValue(args[5], &rpcArgs.Cfg, "Cfg")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Id":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Id, "Id")
			case "Depth":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Depth, "Depth")
			case "Full":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Full, "Full")
			case "Defers":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Defers, "Defers")
			case "Opts":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Opts, "Opts")
			case "Cfg":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Cfg, "Cfg")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("Stacktrace", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["state"] = starlark.NewBuiltin("state", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.StateIn
		var rpcRet service.StateOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.NonBlocking, "NonBlocking")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "NonBlocking":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.NonBlocking, "NonBlocking")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("State", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	r["toggle_breakpoint"] = starlark.NewBuiltin("toggle_breakpoint", func(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := isCancelled(thread); err != nil {
			return starlark.None, decorateError(thread, err)
		}
		var rpcArgs service.ToggleBreakpointIn
		var rpcRet service.ToggleBreakpointOut
		if len(args) > 0 && args[0] != starlark.None {
			err := unmarshalStarlarkValue(args[0], &rpcArgs.Id, "Id")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		if len(args) > 1 && args[1] != starlark.None {
			err := unmarshalStarlarkValue(args[1], &rpcArgs.Name, "Name")
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		for _, kv := range kwargs {
			var err error
			switch kv[0].(starlark.String) {
			case "Id":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Id, "Id")
			case "Name":
				err = unmarshalStarlarkValue(kv[1], &rpcArgs.Name, "Name")
			default:
				err = fmt.Errorf("unknown argument %q", kv[0])
			}
			if err != nil {
				return starlark.None, decorateError(thread, err)
			}
		}
		err := env.ctx.Client().CallAPI("ToggleBreakpoint", &rpcArgs, &rpcRet)
		if err != nil {
			return starlark.None, err
		}
		return env.interfaceToStarlarkValue(rpcRet), nil
	})
	return r
}
