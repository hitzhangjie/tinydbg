package proc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unsafe"

	"github.com/hitzhangjie/tinydbg/pkg/dwarf/op"
	"github.com/hitzhangjie/tinydbg/pkg/dwarf/regnum"
	"github.com/hitzhangjie/tinydbg/pkg/logflags"
)

// readSigtrampgoContext reads runtime.sigtrampgo context at the specified address
func (it *stackIterator) readSigtrampgoContext() (*op.DwarfRegisters, error) {
	logger := logflags.LogDebuggerLogger()
	scope := FrameToScope(it.target, it.mem, it.g, 0, it.frame)
	bi := it.bi

	findvar := func(name string) *Variable {
		vars, _ := scope.Locals(0, name)
		for i := range vars {
			if vars[i].Name == name {
				return vars[i]
			}
		}
		return nil
	}

	deref := func(v *Variable) (uint64, error) {
		v.loadValue(loadSingleValue)
		if v.Unreadable != nil {
			return 0, fmt.Errorf("could not dereference %s: %v", v.Name, v.Unreadable)
		}
		if len(v.Children) < 1 {
			return 0, fmt.Errorf("could not dereference %s (no children?)", v.Name)
		}
		logger.Debugf("%s address is %#x", v.Name, v.Children[0].Addr)
		return v.Children[0].Addr, nil
	}

	getctxaddr := func() (uint64, error) {
		ctxvar := findvar("ctx")
		if ctxvar == nil {
			return 0, errors.New("ctx variable not found")
		}
		addr, err := deref(ctxvar)
		if err != nil {
			return 0, err
		}
		return addr, nil
	}

	if bi.GOOS != "linux" || bi.Arch.Name != "amd64" {
		return nil, errors.New("not implemented")
	}

	addr, err := getctxaddr()
	if err != nil {
		return nil, err
	}

	return sigtrampContextLinuxAMD64(it.mem, addr)
}

func sigtrampContextLinuxAMD64(mem MemoryReader, addr uint64) (*op.DwarfRegisters, error) {
	type stackt struct {
		ss_sp     uint64
		ss_flags  int32
		pad_cgo_0 [4]byte
		ss_size   uintptr
	}

	type mcontext struct {
		r8          uint64
		r9          uint64
		r10         uint64
		r11         uint64
		r12         uint64
		r13         uint64
		r14         uint64
		r15         uint64
		rdi         uint64
		rsi         uint64
		rbp         uint64
		rbx         uint64
		rdx         uint64
		rax         uint64
		rcx         uint64
		rsp         uint64
		rip         uint64
		eflags      uint64
		cs          uint16
		gs          uint16
		fs          uint16
		__pad0      uint16
		err         uint64
		trapno      uint64
		oldmask     uint64
		cr2         uint64
		fpstate     uint64 // pointer
		__reserved1 [8]uint64
	}

	type fpxreg struct {
		significand [4]uint16
		exponent    uint16
		padding     [3]uint16
	}

	type fpstate struct {
		cwd       uint16
		swd       uint16
		ftw       uint16
		fop       uint16
		rip       uint64
		rdp       uint64
		mxcsr     uint32
		mxcr_mask uint32
		_st       [8]fpxreg
		_xmm      [16][4]uint32
		padding   [24]uint32
	}

	type ucontext struct {
		uc_flags     uint64
		uc_link      uint64
		uc_stack     stackt
		uc_mcontext  mcontext
		uc_sigmask   [16]uint64
		__fpregs_mem fpstate
	}

	buf := make([]byte, unsafe.Sizeof(ucontext{}))
	_, err := mem.ReadMemory(buf, addr)
	if err != nil {
		return nil, err
	}
	regs := &(((*ucontext)(unsafe.Pointer(&buf[0]))).uc_mcontext)
	dregs := make([]*op.DwarfRegister, regnum.AMD64MaxRegNum()+1)
	dregs[regnum.AMD64_R8] = op.DwarfRegisterFromUint64(regs.r8)
	dregs[regnum.AMD64_R9] = op.DwarfRegisterFromUint64(regs.r9)
	dregs[regnum.AMD64_R10] = op.DwarfRegisterFromUint64(regs.r10)
	dregs[regnum.AMD64_R11] = op.DwarfRegisterFromUint64(regs.r11)
	dregs[regnum.AMD64_R12] = op.DwarfRegisterFromUint64(regs.r12)
	dregs[regnum.AMD64_R13] = op.DwarfRegisterFromUint64(regs.r13)
	dregs[regnum.AMD64_R14] = op.DwarfRegisterFromUint64(regs.r14)
	dregs[regnum.AMD64_R15] = op.DwarfRegisterFromUint64(regs.r15)
	dregs[regnum.AMD64_Rdi] = op.DwarfRegisterFromUint64(regs.rdi)
	dregs[regnum.AMD64_Rsi] = op.DwarfRegisterFromUint64(regs.rsi)
	dregs[regnum.AMD64_Rbp] = op.DwarfRegisterFromUint64(regs.rbp)
	dregs[regnum.AMD64_Rbx] = op.DwarfRegisterFromUint64(regs.rbx)
	dregs[regnum.AMD64_Rdx] = op.DwarfRegisterFromUint64(regs.rdx)
	dregs[regnum.AMD64_Rax] = op.DwarfRegisterFromUint64(regs.rax)
	dregs[regnum.AMD64_Rcx] = op.DwarfRegisterFromUint64(regs.rcx)
	dregs[regnum.AMD64_Rsp] = op.DwarfRegisterFromUint64(regs.rsp)
	dregs[regnum.AMD64_Rip] = op.DwarfRegisterFromUint64(regs.rip)
	dregs[regnum.AMD64_Rflags] = op.DwarfRegisterFromUint64(regs.eflags)
	dregs[regnum.AMD64_Cs] = op.DwarfRegisterFromUint64(uint64(regs.cs))
	dregs[regnum.AMD64_Gs] = op.DwarfRegisterFromUint64(uint64(regs.gs))
	dregs[regnum.AMD64_Fs] = op.DwarfRegisterFromUint64(uint64(regs.fs))
	return op.NewDwarfRegisters(0, dregs, binary.LittleEndian, regnum.AMD64_Rip, regnum.AMD64_Rsp, regnum.AMD64_Rbp, 0), nil
}
