package rpccommon

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"reflect"
	"runtime"
	"sync"

	"github.com/hitzhangjie/tinydbg/pkg/logflags"
	"github.com/hitzhangjie/tinydbg/service"
	"github.com/hitzhangjie/tinydbg/service/api"
	"github.com/hitzhangjie/tinydbg/service/debugger"
	"github.com/hitzhangjie/tinydbg/service/rpc2"
)

//go:generate go run ../../_scripts/gen-suitablemethods.go suitablemethods

// ServerImpl implements a JSON-RPC server that can switch between two
// versions of the API.
type ServerImpl struct {
	// config is all the information necessary to start the debugger and server.
	config *service.Config
	// listener is used to serve JSON-RPC.
	listener net.Listener
	// stopChan is used to stop the listener goroutine.
	stopChan chan struct{}
	// debugger is the debugger service.
	debugger *debugger.Debugger
	// s2 is APIv2 server.
	s2 *rpc2.RPCServer
	// maps of served methods
	methodMap map[string]*methodType
	log       logflags.Logger
}

type RPCCallback struct {
	s              *ServerImpl
	sending        *sync.Mutex
	codec          rpc.ServerCodec
	req            rpc.Request
	setupDone      chan struct{}
	disconnectChan chan struct{}
}

var _ service.RPCCallback = &RPCCallback{}

// RPCServer implements the RPC method calls common to all versions of the API.
type RPCServer struct {
	s *ServerImpl
}

type methodType struct {
	method      reflect.Value
	ArgType     reflect.Type
	ReplyType   reflect.Type
	Synchronous bool
}

// NewServer creates a new RPCServer.
func NewServer(config *service.Config) *ServerImpl {
	logger := logflags.RPCLogger()
	if config.Debugger.Foreground {
		// Print listener address
		logflags.WriteAPIListeningMessage(config.Listener.Addr())
		logger.Debug("API server pid = ", os.Getpid())
	}
	return &ServerImpl{
		config:   config,
		listener: config.Listener,
		stopChan: make(chan struct{}),
		log:      logger,
	}
}

// Stop stops the JSON-RPC server.
func (s *ServerImpl) Stop() error {
	s.log.Debug("stopping")
	close(s.stopChan)
	if s.config.AcceptMulti {
		s.listener.Close()
	}
	if s.debugger.IsRunning() {
		s.debugger.Command(&api.DebuggerCommand{Name: api.Halt}, nil, nil)
	}
	kill := s.config.Debugger.AttachPid == 0
	return s.debugger.Detach(kill)
}

// Run starts a debugger and exposes it with an JSON-RPC server. The debugger
// itself can be stopped with the `detach` API.
func (s *ServerImpl) Run() error {
	var err error

	// Create and start the debugger
	config := s.config.Debugger
	if s.debugger, err = debugger.New(&config, s.config.ProcessArgs); err != nil {
		return err
	}

	s.s2 = rpc2.NewServer(s.config, s.debugger)
	s.methodMap = make(map[string]*methodType)

	registerMethods(s.s2, s.methodMap)

	go func() {
		defer s.listener.Close()
		for {
			c, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.stopChan:
					// We were supposed to exit, do nothing and return
					return
				default:
					panic(err)
				}
			}

			go s.serveConnection(c)
			if !s.config.AcceptMulti {
				break
			}
		}
	}()
	return nil
}

type bufReadWriteCloser struct {
	*bufio.Reader
	io.WriteCloser
}

func (s *ServerImpl) serveConnection(c io.ReadWriteCloser) {
	conn := &bufReadWriteCloser{bufio.NewReader(c), c}
	s.log.Debugf("serving JSON-RPC on new connection")
	go s.serveJSONCodec(conn)
}

func (s *ServerImpl) serveJSONCodec(conn io.ReadWriteCloser) {
	clientDisconnectChan := make(chan struct{})
	defer func() {
		close(clientDisconnectChan)
		if !s.config.AcceptMulti && s.config.DisconnectChan != nil {
			close(s.config.DisconnectChan)
		}
	}()

	sending := new(sync.Mutex)
	codec := jsonrpc.NewServerCodec(conn)
	var req rpc.Request
	var resp rpc.Response
	for {
		req = rpc.Request{}
		err := codec.ReadRequestHeader(&req)
		if err != nil {
			if err != io.EOF {
				s.log.Error("rpc:", err)
			}
			break
		}

		mtype, ok := s.methodMap[req.ServiceMethod]
		if !ok {
			s.log.Errorf("rpc: can't find method %s", req.ServiceMethod)
			s.sendResponse(sending, &req, &rpc.Response{}, nil, codec, fmt.Sprintf("unknown method: %s", req.ServiceMethod))
			continue
		}

		var argv, replyv reflect.Value

		// Decode the argument value.
		argIsValue := false // if true, need to indirect before calling.
		if mtype.ArgType.Kind() == reflect.Ptr {
			argv = reflect.New(mtype.ArgType.Elem())
		} else {
			argv = reflect.New(mtype.ArgType)
			argIsValue = true
		}
		// argv guaranteed to be a pointer now.
		if err = codec.ReadRequestBody(argv.Interface()); err != nil {
			return
		}
		if argIsValue {
			argv = argv.Elem()
		}

		if mtype.Synchronous {
			if logflags.LogRPC() {
				argvbytes, _ := json.Marshal(argv.Interface())
				s.log.Debugf("<- %s(%T%s)", req.ServiceMethod, argv.Interface(), argvbytes)
			}
			replyv = reflect.New(mtype.ReplyType.Elem())
			function := mtype.method
			var returnValues []reflect.Value
			var errInter interface{}
			func() {
				defer func() {
					if ierr := recover(); ierr != nil {
						errInter = newInternalError(ierr, 2)
					}
				}()
				returnValues = function.Call([]reflect.Value{argv, replyv})
				errInter = returnValues[0].Interface()
			}()

			errmsg := ""
			if errInter != nil {
				errmsg = errInter.(error).Error()
			}
			resp = rpc.Response{}
			if logflags.LogRPC() {
				replyvbytes, _ := json.Marshal(replyv.Interface())
				s.log.Debugf("-> %T%s error: %q", replyv.Interface(), replyvbytes, errmsg)
			}
			s.sendResponse(sending, &req, &resp, replyv.Interface(), codec, errmsg)
			if req.ServiceMethod == "RPCServer.Detach" && s.config.DisconnectChan != nil {
				close(s.config.DisconnectChan)
				s.config.DisconnectChan = nil
			}
		} else {
			if logflags.LogRPC() {
				argvbytes, _ := json.Marshal(argv.Interface())
				s.log.Debugf("(async %d) <- %s(%T%s)", req.Seq, req.ServiceMethod, argv.Interface(), argvbytes)
			}
			function := mtype.method
			ctl := &RPCCallback{s, sending, codec, req, make(chan struct{}), clientDisconnectChan}
			go func() {
				defer func() {
					if ierr := recover(); ierr != nil {
						ctl.Return(nil, newInternalError(ierr, 2))
					}
				}()
				function.Call([]reflect.Value{argv, reflect.ValueOf(ctl)})
			}()
			<-ctl.setupDone
		}
	}
	codec.Close()
}

// A value sent as a placeholder for the server's response value when the server
// receives an invalid request. It is never decoded by the client since the Response
// contains an error when it is used.
var invalidRequest = struct{}{}

func (s *ServerImpl) sendResponse(sending *sync.Mutex, req *rpc.Request, resp *rpc.Response, reply interface{}, codec rpc.ServerCodec, errmsg string) {
	resp.ServiceMethod = req.ServiceMethod
	if errmsg != "" {
		resp.Error = errmsg
		reply = invalidRequest
	}
	resp.Seq = req.Seq
	sending.Lock()
	defer sending.Unlock()
	err := codec.WriteResponse(resp, reply)
	if err != nil {
		s.log.Error("writing response:", err)
	}
}

func (cb *RPCCallback) Return(out interface{}, err error) {
	select {
	case <-cb.setupDone:
		// already closed
	default:
		close(cb.setupDone)
	}
	errmsg := ""
	if err != nil {
		errmsg = err.Error()
	}
	var resp rpc.Response
	if logflags.LogRPC() {
		outbytes, _ := json.Marshal(out)
		cb.s.log.Debugf("(async %d) -> %T%s error: %q", cb.req.Seq, out, outbytes, errmsg)
	}

	if cb.hasDisconnected() {
		return
	}

	cb.s.sendResponse(cb.sending, &cb.req, &resp, out, cb.codec, errmsg)
}

func (cb *RPCCallback) DisconnectChan() chan struct{} {
	return cb.disconnectChan
}

func (cb *RPCCallback) hasDisconnected() bool {
	select {
	case <-cb.disconnectChan:
		return true
	default:
	}

	return false
}

func (cb *RPCCallback) SetupDoneChan() chan struct{} {
	return cb.setupDone
}

type internalError struct {
	Err   interface{}
	Stack []internalErrorFrame
}

type internalErrorFrame struct {
	Pc   uintptr
	Func string
	File string
	Line int
}

func newInternalError(ierr interface{}, skip int) *internalError {
	r := &internalError{ierr, nil}
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fname := "<unknown>"
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			fname = fn.Name()
		}
		r.Stack = append(r.Stack, internalErrorFrame{pc, fname, file, line})
	}
	return r
}

func (err *internalError) Error() string {
	var out bytes.Buffer
	fmt.Fprintf(&out, "Internal debugger error: %v\n", err.Err)
	for _, frame := range err.Stack {
		fmt.Fprintf(&out, "%s (%#x)\n\t%s:%d\n", frame.Func, frame.Pc, frame.File, frame.Line)
	}
	return out.String()
}

func registerMethods(s *rpc2.RPCServer, methods map[string]*methodType) {
	defer func() {
		for name, method := range methods {
			mtype := method.method.Type()
			if mtype.NumIn() != 2 {
				panic(fmt.Errorf("wrong number of inputs for method %s (%d)", name, mtype.NumIn()))
			}
			method.ArgType = mtype.In(0)
			method.ReplyType = mtype.In(1)
			method.Synchronous = method.ReplyType.String() != "service.RPCCallback"
		}
	}()
	methods["RPCServer.AmendBreakpoint"] = &methodType{method: reflect.ValueOf(s.AmendBreakpoint)}
	methods["RPCServer.Ancestors"] = &methodType{method: reflect.ValueOf(s.Ancestors)}
	methods["RPCServer.AttachedToExistingProcess"] = &methodType{method: reflect.ValueOf(s.AttachedToExistingProcess)}
	methods["RPCServer.BuildID"] = &methodType{method: reflect.ValueOf(s.BuildID)}
	methods["RPCServer.CancelNext"] = &methodType{method: reflect.ValueOf(s.CancelNext)}
	methods["RPCServer.ClearBreakpoint"] = &methodType{method: reflect.ValueOf(s.ClearBreakpoint)}
	methods["RPCServer.Command"] = &methodType{method: reflect.ValueOf(s.Command)}
	methods["RPCServer.CreateBreakpoint"] = &methodType{method: reflect.ValueOf(s.CreateBreakpoint)}
	methods["RPCServer.CreateWatchpoint"] = &methodType{method: reflect.ValueOf(s.CreateWatchpoint)}
	methods["RPCServer.Detach"] = &methodType{method: reflect.ValueOf(s.Detach)}
	methods["RPCServer.Disassemble"] = &methodType{method: reflect.ValueOf(s.Disassemble)}
	methods["RPCServer.DumpCancel"] = &methodType{method: reflect.ValueOf(s.DumpCancel)}
	methods["RPCServer.DumpStart"] = &methodType{method: reflect.ValueOf(s.DumpStart)}
	methods["RPCServer.DumpWait"] = &methodType{method: reflect.ValueOf(s.DumpWait)}
	methods["RPCServer.Eval"] = &methodType{method: reflect.ValueOf(s.Eval)}
	methods["RPCServer.ExamineMemory"] = &methodType{method: reflect.ValueOf(s.ExamineMemory)}
	methods["RPCServer.FindLocation"] = &methodType{method: reflect.ValueOf(s.FindLocation)}
	methods["RPCServer.FollowExec"] = &methodType{method: reflect.ValueOf(s.FollowExec)}
	methods["RPCServer.FollowExecEnabled"] = &methodType{method: reflect.ValueOf(s.FollowExecEnabled)}
	methods["RPCServer.FunctionReturnLocations"] = &methodType{method: reflect.ValueOf(s.FunctionReturnLocations)}
	methods["RPCServer.GetBreakpoint"] = &methodType{method: reflect.ValueOf(s.GetBreakpoint)}
	methods["RPCServer.GetThread"] = &methodType{method: reflect.ValueOf(s.GetThread)}
	methods["RPCServer.GuessSubstitutePath"] = &methodType{method: reflect.ValueOf(s.GuessSubstitutePath)}
	methods["RPCServer.IsMulticlient"] = &methodType{method: reflect.ValueOf(s.IsMulticlient)}
	methods["RPCServer.ListBreakpoints"] = &methodType{method: reflect.ValueOf(s.ListBreakpoints)}
	methods["RPCServer.ListDynamicLibraries"] = &methodType{method: reflect.ValueOf(s.ListDynamicLibraries)}
	methods["RPCServer.ListFunctionArgs"] = &methodType{method: reflect.ValueOf(s.ListFunctionArgs)}
	methods["RPCServer.ListFunctions"] = &methodType{method: reflect.ValueOf(s.ListFunctions)}
	methods["RPCServer.ListGoroutines"] = &methodType{method: reflect.ValueOf(s.ListGoroutines)}
	methods["RPCServer.ListLocalVars"] = &methodType{method: reflect.ValueOf(s.ListLocalVars)}
	methods["RPCServer.ListPackageVars"] = &methodType{method: reflect.ValueOf(s.ListPackageVars)}
	methods["RPCServer.ListPackagesBuildInfo"] = &methodType{method: reflect.ValueOf(s.ListPackagesBuildInfo)}
	methods["RPCServer.ListRegisters"] = &methodType{method: reflect.ValueOf(s.ListRegisters)}
	methods["RPCServer.ListSources"] = &methodType{method: reflect.ValueOf(s.ListSources)}
	methods["RPCServer.ListTargets"] = &methodType{method: reflect.ValueOf(s.ListTargets)}
	methods["RPCServer.ListThreads"] = &methodType{method: reflect.ValueOf(s.ListThreads)}
	methods["RPCServer.ListTypes"] = &methodType{method: reflect.ValueOf(s.ListTypes)}
	methods["RPCServer.ProcessPid"] = &methodType{method: reflect.ValueOf(s.ProcessPid)}
	methods["RPCServer.Restart"] = &methodType{method: reflect.ValueOf(s.Restart)}
	methods["RPCServer.Set"] = &methodType{method: reflect.ValueOf(s.Set)}
	methods["RPCServer.Stacktrace"] = &methodType{method: reflect.ValueOf(s.Stacktrace)}
	methods["RPCServer.State"] = &methodType{method: reflect.ValueOf(s.State)}
	methods["RPCServer.StopRecording"] = &methodType{method: reflect.ValueOf(s.StopRecording)}
	methods["RPCServer.ToggleBreakpoint"] = &methodType{method: reflect.ValueOf(s.ToggleBreakpoint)}
}
