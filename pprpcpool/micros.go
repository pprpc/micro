package pprpcpool

import (
	"context"
	"fmt"

	"xcthings.com/hjyz/cache"
	"xcthings.com/micro/svc"
	"xcthings.com/pprpc"
	"xcthings.com/pprpc/packets"
)

// ClientPool .
type ClientPool struct {
	Name string
	*RPCCliPool
}

// MicroClientConn micro service conn
type MicroClientConn struct {
	Micros   []ClientPool
	service  *pprpc.Service
	regCache *cache.Cache
}

// NewMicroClientConn new micro service client connection.
func NewMicroClientConn(s *pprpc.Service) (mcc *MicroClientConn) {
	mcc = new(MicroClientConn)
	mcc.regCache = cache.NewCache(10000)
	mcc.service = s
	return
}

// AddMicro add micro service pool
func (m *MicroClientConn) AddMicro(ms string) (err error) {
	cliPool := NewRPCCliPool()
	cliPool.Service = m.service

	var cp ClientPool
	cp.Name = ms
	cp.RPCCliPool = cliPool
	m.Micros = append(m.Micros, cp)
	return
}

// Invoke rpc call
func (m *MicroClientConn) Invoke(ctx context.Context, ms string, cmdid uint64, req interface{}) (pkg *packets.CmdPacket, resp interface{}, err error) {
	for _, v := range m.Micros {
		if v.Name == ms {
			pkg, resp, err = v.Invoke(ctx, cmdid, req)
			return
		}
	}
	err = fmt.Errorf("No microservices found: %s", ms)
	return
}

// InvokeServerID call invoke by server id
func (m *MicroClientConn) InvokeServerID(ctx context.Context, ms, serverID string, cmdid uint64, req interface{}) (pkg *packets.CmdPacket, resp interface{}, err error) {
	for _, v := range m.Micros {
		if v.Name == ms {
			pkg, resp, err = v.InvokeByServerID(ctx, serverID, cmdid, req)
			return
		}
	}
	err = fmt.Errorf("No microservices found: %s", ms)
	return
}

// AddHost add micro service host
func (m *MicroClientConn) AddHost(key string, vrs svc.ValueRegService) (err error) {
	for _, v := range m.Micros {
		if v.Name == vrs.Name {
			url, e := svc.GetTCPURL(vrs)
			if e != nil {
				err = fmt.Errorf("svc.GetTCPURL(), %s(%v)", err, vrs)
				return
			}
			err = v.AddHost(url)
			if err != nil {
				err = fmt.Errorf("microClientInit, AddHost(%s), error: %s", url, err)
			} else {
				m.regCache.AddORUpdate(key, vrs)
			}
			return
		}
	}
	return
}

// DelHost del micro service host.
func (m *MicroClientConn) DelHost(key string) (err error) {
	var url string
	v, e := m.regCache.Get(key)
	if e != nil {
		err = fmt.Errorf("g.RegCache.Get(%s), %s", key, e)
		return
	}
	vrs := v.(svc.ValueRegService)

	for _, v := range m.Micros {
		if v.Name == vrs.Name {
			url, err = svc.GetTCPURL(vrs)
			if err != nil {
				err = fmt.Errorf("svc.GetTCPURL(), %s(%v)", err, vrs)
			}
			err = v.RPCCliPool.DelHost(url)
			if err != nil {
				err = fmt.Errorf("DelHost(%s), error: %s", url, err)
			} else {
				m.regCache.Del(key)
			}
			return
		}
	}
	return
}
