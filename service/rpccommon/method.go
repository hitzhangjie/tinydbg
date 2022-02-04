package rpccommon

import "reflect"

type methodType struct {
	method      reflect.Method
	Rcvr        reflect.Value
	ArgType     reflect.Type
	ReplyType   reflect.Type
	Synchronous bool
}
