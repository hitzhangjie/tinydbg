package main

import (
	"fmt"
	"github.com/hitzhangjie/dlv/_fixtures/internal/dir.io"
	"github.com/hitzhangjie/dlv/_fixtures/internal/dir.io/io.io"
	"runtime"
)

func main() {
	var iface interface{} = &dirio.SomeType{}
	var iface2 interface{} = &ioio.SomeOtherType{}
	runtime.Breakpoint()
	fmt.Println(iface, iface2)
}
