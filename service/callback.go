package service

import (
	"encoding/json"
	"net/rpc"
	"sync"

	"github.com/hitzhangjie/dlv/pkg/log"
)

// RPCCallback is used by RPC methods to return their result asynchronously.
type RPCCallback interface {
	// Return rpc method should call cb.Return to notifies the rpc server to
	// return the result to rpc client. This method should close the setup
	// done chan, so <-SetupDoneChan() will not block any more.
	Return(out interface{}, err error)

	// SetupDone returns a channel that should be closed to signal that the
	// asynchornous method has completed setup and the server is ready to
	// receive other requests.
	SetupDoneChan() chan struct{}
}

// rpcCallback implements the service.RPCCallback interface
type rpcCallback struct {
	s         *server
	sending   *sync.Mutex
	codec     rpc.ServerCodec
	req       rpc.Request
	setupDone chan struct{}
}

// Ensures rpcCallback satisfies the service.RPCCallback interface
var _ RPCCallback = &rpcCallback{}

func (cb *rpcCallback) Return(out interface{}, err error) {
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
	outbytes, _ := json.Marshal(out)
	log.Debug("(async %d) -> %T%s error: %q", cb.req.Seq, out, outbytes, errmsg)

	cb.s.sendResponse(cb.sending, &cb.req, &resp, out, cb.codec, errmsg)
}

func (cb *rpcCallback) SetupDoneChan() chan struct{} {
	return cb.setupDone
}
