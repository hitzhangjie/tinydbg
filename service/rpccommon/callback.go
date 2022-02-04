package rpccommon

import (
	"encoding/json"
	"net/rpc"
	"sync"

	"github.com/hitzhangjie/dlv/pkg/log"
	"github.com/hitzhangjie/dlv/service"
)

// rpcCallback implements the service.RPCCallback interface
type rpcCallback struct {
	s         *server
	sending   *sync.Mutex
	codec     rpc.ServerCodec
	req       rpc.Request
	setupDone chan struct{}
}

var _ service.RPCCallback = &rpcCallback{}

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
