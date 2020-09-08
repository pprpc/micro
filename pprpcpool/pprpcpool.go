package pprpcpool

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"xcthings.com/hjyz/logs"
	"xcthings.com/pprpc"
	"xcthings.com/pprpc/packets"
	"xcthings.com/pprpc/pptcp"
	"xcthings.com/pprpc/sess"
)

// ClientConnInfo client info
type ClientConnInfo struct {
	CPU []float32 //
	Mem []float32 // total, free, 比例
	Sys []float32 // 5, 10, 15

	urladdr string
	url     *url.URL
	cli     *pprpc.TCPCliConn
}

// RPCCliPool PPRPC conn pool
type RPCCliPool struct {
	//clis *sync.Map // ClientConnInfo
	clis           *sess.Sessions
	Service        *pprpc.Service
	totalReq       uint32
	addrs          []string
	mu             sync.Mutex
	WriteTimeoutMs int
}

// NewRPCCliPool create rpc client conn pool
func NewRPCCliPool() *RPCCliPool {
	_t := new(RPCCliPool)
	_t.clis = sess.NewSessions(8000)
	_t.mu = sync.Mutex{}
	_t.WriteTimeoutMs = 3000

	return _t
}

// SetWriteTimeout ms
func (r *RPCCliPool) SetWriteTimeout(ms int) (err error) {
	if r.Service == nil {
		err = fmt.Errorf("SetWriteTimeout, error: not set Service")
		return
	}
	if ms < 500 && ms > 100000 {
		err = fmt.Errorf("Out of range: 500-100000")
		return
	}
	r.WriteTimeoutMs = ms
	return
}

// AddHost .
func (r *RPCCliPool) AddHost(addr string) (err error) {
	if r.Service == nil {
		err = fmt.Errorf("AddHost, error: not set Service")
		return
	}
	uri := fmt.Sprintf("%s", addr)
	u, e := url.ParseRequestURI(uri)
	if e != nil {
		err = fmt.Errorf("url.ParseRequestURI(%s), error: %s", uri, e)
		return
	}
	var err1 error
	conn, e := pprpc.Dail(u, nil, r.Service, 5*time.Second, nil)
	if e != nil {
		err1 = fmt.Errorf("pprpc.Dail(), error: %s", e)

	}
	conn.SyncWriteTimeoutMs = r.WriteTimeoutMs
	err = r.addHost(addr, conn)
	if err == nil && err1 != nil {
		err = err1
	} else if err != nil && err1 != nil {
		err = fmt.Errorf("%s; r.addHost(addr, conn), %s", err1, err)
	}

	return
}

// DelHost .
func (r *RPCCliPool) DelHost(addr string) (err error) {
	v, e := r.clis.Get(addr)
	if e != nil {
		err = fmt.Errorf("Load(%s), %s", addr, e)
		return
	}
	v.(*pprpc.TCPCliConn).Close()
	r.delHost(addr)

	return
}

// Invoke .
func (r *RPCCliPool) Invoke(ctx context.Context, cmdid uint64, req interface{}) (pkg *packets.CmdPacket, resp interface{}, err error) {
	var cli *pprpc.TCPCliConn
	cli, err = r.GetCli()
	if cli == nil || err != nil {
		return
	}
	// debugs
	logs.Logger.Debugf("addrs: %v, client info: %s.", r.addrs, cli.String())

	pkg, resp, err = cli.Invoke(ctx, cmdid, req)

	return
}

// InvokeAsync .
func (r *RPCCliPool) InvokeAsync(ctx context.Context, cmdid uint64, req interface{}) (err error) {
	var cli *pprpc.TCPCliConn
	cli, err = r.GetCli()
	if cli == nil || err != nil {
		return
	}
	err = cli.InvokeAsync(ctx, cmdid, req)

	return
}

// lbs .
func (r *RPCCliPool) lbs() {

}

// GetTotalReq .
func (r *RPCCliPool) GetTotalReq() uint32 {
	return atomic.LoadUint32(&r.totalReq)
}

func (r *RPCCliPool) getCli() (cli *pprpc.TCPCliConn, err error) {
	v := atomic.LoadUint32(&r.totalReq) + 1
	atomic.StoreUint32(&r.totalReq, v)

	m := v % uint32(len(r.addrs))
	key := r.addrs[m]
	c, e := r.clis.Get(key)
	if e != nil {
		err = e
		return
	}
	cli = c.(*pprpc.TCPCliConn)
	return
}

// GetCli .
func (r *RPCCliPool) GetCli() (cli *pprpc.TCPCliConn, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	len := len(r.addrs)
	for i := 0; i < len; i++ {
		cli, err = r.getCli()
		if err != nil {
			continue
		}
		s, e := cli.GetState()
		if e != nil || s != pptcp.StateConnected {
			err = fmt.Errorf("Connection status is incorrect")
			continue
		} else {
			return
		}
	}
	if cli == nil {
		err = fmt.Errorf("No microservices found")
	}
	return
}

func (r *RPCCliPool) addHost(addr string, conn *pprpc.TCPCliConn) (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	isExist := false
	for _, v := range r.addrs {
		if v == addr {
			isExist = true
			break
		}
	}
	if isExist == false {
		r.addrs = append(r.addrs, addr)
	}
	_, err = r.clis.Push(addr, conn)
	return
}

func (r *RPCCliPool) delHost(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var i int
	var v string
	for i, v = range r.addrs {
		if v == addr {
			if i+1 <= len(r.addrs) {
				r.addrs = append(r.addrs[:i], r.addrs[i+1:]...)
			} else {
				r.addrs = r.addrs[:i]
			}
			break
		}
	}
	r.clis.Remove(addr)
	return
}

// InvokeByServerID .
func (r *RPCCliPool) InvokeByServerID(ctx context.Context, serverID string, cmdid uint64, req interface{}) (pkg *packets.CmdPacket, resp interface{}, err error) {
	var cli *pprpc.TCPCliConn
	cli, err = r.GetCliByServerID(serverID)
	if cli == nil || err != nil {
		return
	}
	pkg, resp, err = cli.Invoke(ctx, cmdid, req)

	return
}

// InvokeAsyncByServerID .
func (r *RPCCliPool) InvokeAsyncByServerID(ctx context.Context, serverID string, cmdid uint64, req interface{}) (err error) {
	var cli *pprpc.TCPCliConn
	cli, err = r.GetCliByServerID(serverID)
	if cli == nil || err != nil {
		return
	}
	err = cli.InvokeAsync(ctx, cmdid, req)

	return
}

// GetCliByServerID .
func (r *RPCCliPool) GetCliByServerID(serverID string) (cli *pprpc.TCPCliConn, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, v := range r.addrs {
		u, e := url.ParseRequestURI(v)
		if e != nil {
			err = fmt.Errorf("url.ParseRequestURI(%s), %s", v, e)
			return
		}
		if serverID == u.Hostname() {
			c, e := r.clis.Get(v)
			if e != nil {
				err = fmt.Errorf("r.clis.Get(%s), %s", v, e)
				return
			}
			cli = c.(*pprpc.TCPCliConn)
			return
		}
	}
	if cli == nil {
		err = fmt.Errorf("No microservices found(server_id): %s", serverID)
	}
	return
}
