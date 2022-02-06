package service

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
	"unicode"
	"unicode/utf8"

	"github.com/hitzhangjie/dlv/pkg/log"
	"github.com/hitzhangjie/dlv/service/api"
	"github.com/hitzhangjie/dlv/service/debugger"
)

// Server represents a server for a remote client to connect to.
type Server interface {
	Run() error
	Stop() error
}

// server implements a JSON-RPC server.
type server struct {
	config     *Config                // all information necessary to start the debugger and server
	listener   net.Listener           // serve HTTP
	stopChan   chan struct{}          // stop the listener goroutine
	debugger   *debugger.Debugger     // debugger service
	rpcServer  *RPCServer             // APIv2 server
	methodMaps map[string]*methodType // maps of served methods
}

type methodType struct {
	method      reflect.Method
	Rcvr        reflect.Value
	ArgType     reflect.Type
	ReplyType   reflect.Type
	Synchronous bool
}

// NewServer creates a new server which serves debugging RPC requests.
func NewServer(config *Config) *server {
	if config.DebuggerConfig.Foreground {
		log.Info("listen address: %s", config.Listener.Addr())
		log.Debug("API server pid = %d", os.Getpid())
	}
	return &server{
		config:   config,
		listener: config.Listener,
		stopChan: make(chan struct{}),
	}
}

// Run starts a debugger and exposes it with an HTTP server. The debugger
// itself can be stopped with the `detach` API. Run blocks until the HTTP
// server stops.
func (s *server) Run() error {
	var err error
	// Create and start the debugger
	config := s.config.DebuggerConfig
	if s.debugger, err = debugger.New(&config, s.config.ProcessArgs); err != nil {
		return err
	}

	s.rpcServer = NewRPCServer(s.config, s.debugger)

	s.methodMaps = make(map[string]*methodType)
	suitableMethods(s.rpcServer, s.methodMaps)

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

			go s.serveConnectionDemux(c)
			if !s.config.AcceptMulti {
				break
			}
		}
	}()
	return nil
}

// Stop stops the JSON-RPC server.
func (s *server) Stop() error {
	log.Debug("stopping")
	close(s.stopChan)
	if s.config.AcceptMulti {
		s.listener.Close()
	}
	if s.debugger.IsRunning() {
		s.debugger.Command(&api.DebuggerCommand{Name: api.Halt}, nil)
	}
	// if tracee is launched by tracer, kill it
	kill := s.config.DebuggerConfig.AttachPid == 0
	return s.debugger.Detach(kill)
}

func (s *server) serveConnectionDemux(c io.ReadWriteCloser) {
	conn := &bufReadWriteCloser{bufio.NewReader(c), c}
	b, err := conn.Peek(1)
	if err != nil {
		log.Warn("error determining new connection protocol: %v", err)
		return
	}
	if b[0] == 'C' { // C is for DAP's Content-Length
		panic("serving DAP on new connection...not supported")
	} else {
		log.Debug("serving JSON-RPC on new connection")
		go s.serveJSONCodec(conn)
	}
}

type bufReadWriteCloser struct {
	*bufio.Reader
	io.WriteCloser
}

// Precompute the reflect type for error.  Can't use error directly
// because Typeof takes an empty interface value.  This is annoying.
var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

// Is this an exported - upper case - name?
func isExported(name string) bool {
	ch, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(ch)
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

// Fills methods map with the methods of receiver that should be made
// available through the RPC interface.
// These are all the public methods of rcvr that have one of those
// two signatures:
//  func (rcvr ReceiverType) Method(in InputType, out *ReplyType) error
//  func (rcvr ReceiverType) Method(in InputType, cb service.RPCCallback)
func suitableMethods(rcvr interface{}, methods map[string]*methodType) {
	typ := reflect.TypeOf(rcvr)
	rcvrv := reflect.ValueOf(rcvr)
	sname := reflect.Indirect(rcvrv).Type().Name()
	if sname == "" {
		log.Debug("rpcv2.Register: no service name for type %s", typ)
		return
	}
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mname := method.Name
		mtype := method.Type
		// method must be exported
		if method.PkgPath != "" {
			continue
		}
		// Method needs three ins: (receive, *args, *reply) or (receiver, *args, *rpcCallback)
		if mtype.NumIn() != 3 {
			log.Warn("method %s has wrong number of ins: %d", mname, mtype.NumIn())
			continue
		}
		// First arg need not be a pointer.
		argType := mtype.In(1)
		if !isExportedOrBuiltinType(argType) {
			log.Warn("%s argument type not exported: %s", mname, argType)
			continue
		}

		replyType := mtype.In(2)
		synchronous := replyType.String() != "service.RPCCallback"

		if synchronous {
			// Second arg must be a pointer.
			if replyType.Kind() != reflect.Ptr {
				log.Warn("method %s reply type not a pointer: %s", mname, replyType)
				continue
			}
			// Reply type must be exported.
			if !isExportedOrBuiltinType(replyType) {
				log.Warn("method %s reply type not exported: %s", mname, replyType)
				continue
			}

			// Method needs one out.
			if mtype.NumOut() != 1 {
				log.Warn("method %s has wrong number of outs: %d", mname, mtype.NumOut())
				continue
			}
			// The return type of the method must be error.
			if returnType := mtype.Out(0); returnType != typeOfError {
				log.Warn("method %s returns %s not error", mname, returnType)
				continue
			}
		} else if mtype.NumOut() != 0 {
			// Method needs zero outs.
			log.Warn("method %s has wrong number of outs: %d", mname, mtype.NumOut())
			continue
		}
		methods[sname+"."+mname] = &methodType{method: method, ArgType: argType, ReplyType: replyType, Synchronous: synchronous, Rcvr: rcvrv}
	}
}

func (s *server) serveJSONCodec(conn io.ReadWriteCloser) {
	defer func() {
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
				log.Error("rpcv2: %v", err)
			}
			break
		}

		mtype, ok := s.methodMaps[req.ServiceMethod]
		if !ok {
			log.Error("rpcv2: can't find method: %s", req.ServiceMethod)
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
			argvbytes, _ := json.Marshal(argv.Interface())
			log.Debug("<- %s(%T%s)", req.ServiceMethod, argv.Interface(), argvbytes)

			replyv = reflect.New(mtype.ReplyType.Elem())
			function := mtype.method.Func
			var returnValues []reflect.Value
			var errInter interface{}
			func() {
				defer func() {
					if ierr := recover(); ierr != nil {
						errInter = newInternalError(ierr, 2)
					}
				}()
				returnValues = function.Call([]reflect.Value{mtype.Rcvr, argv, replyv})
				errInter = returnValues[0].Interface()
			}()

			errmsg := ""
			if errInter != nil {
				errmsg = errInter.(error).Error()
			}
			resp = rpc.Response{}
			replyvbytes, _ := json.Marshal(replyv.Interface())
			log.Debug("-> %T%s error: %q", replyv.Interface(), replyvbytes, errmsg)

			s.sendResponse(sending, &req, &resp, replyv.Interface(), codec, errmsg)
			if req.ServiceMethod == "RPCServer.Detach" && s.config.DisconnectChan != nil {
				close(s.config.DisconnectChan)
				s.config.DisconnectChan = nil
			}
		} else {
			argvbytes, _ := json.Marshal(argv.Interface())
			log.Debug("(async %d) <- %s(%T%s)", req.Seq, req.ServiceMethod, argv.Interface(), argvbytes)

			function := mtype.method.Func
			callback := &rpcCallback{s, sending, codec, req, make(chan struct{})}
			go func() {
				defer func() {
					if err := recover(); err != nil {
						callback.Return(nil, newInternalError(err, 2))
					}
				}()
				function.Call([]reflect.Value{mtype.Rcvr, argv, reflect.ValueOf(callback)})
			}()
			<-callback.setupDone
		}
	}
	codec.Close()
}

// A value sent as a placeholder for the server's response value when the server
// receives an invalid request. It is never decoded by the client since the Response
// contains an error when it is used.
var invalidRequest = struct{}{}

func (s *server) sendResponse(sending *sync.Mutex, req *rpc.Request, resp *rpc.Response, reply interface{}, codec rpc.ServerCodec, errmsg string) {
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
		log.Error("writing response: %v", err)
	}
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
