package svc

// 服务注册与发现

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	//"github.com/coreos/etcd/clientv3"
	"go.etcd.io/etcd/clientv3"
	"xcthings.com/hjyz/cache"
	"xcthings.com/hjyz/common"
	"xcthings.com/hjyz/logs"
)

type WatcherCB func(action, key, value string)

// Agent .
type Agent struct {
	ctx       context.Context
	ctxCancel context.CancelFunc

	leaseid   clientv3.LeaseID
	client    *clientv3.Client
	key       string
	value     ValueRegService
	leaseTime int64
}

// NewAgent create service
func NewAgent(info ValueRegService, leaseTime int64, endpoints []string) (sv *Agent, err error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	if leaseTime < 5 {
		leaseTime = 5
	} else if leaseTime > 60 {
		leaseTime = 60
	}

	sv = new(Agent)
	sv.value = info
	sv.client = cli
	sv.leaseTime = leaseTime
	sv.ctx, sv.ctxCancel = context.WithCancel(context.Background())

	// /register/region/msname/lanip
	sv.key = fmt.Sprintf("/register/%s/%s/%s", info.Region, info.Name, info.LanIP)

	return
}

// Start start register
func (s *Agent) Start() {
start:
	ctx, _ := context.WithTimeout(context.TODO(), 5*time.Second)
	ch, err := s.keepAlive(context.TODO())
	if err != nil {
		logs.Logger.Errorf("s.keepAlive(), %s, sleep 5sec restart.", err)
		common.Sleep(5)
		goto start
	}

	for {
		select {
		case <-s.ctx.Done():
			s.revoke(ctx)
			logs.Logger.Error("Agent.Start(), s.ctx.Done().")
			return
		case <-s.client.Ctx().Done():
			logs.Logger.Error("Agent.Start(), closed.")
			return
		case _, ok := <-ch:
			if !ok {
				logs.Logger.Warnf("keep alive closed, key: %s, restart KeepAlive.", s.key)
				err := s.revoke(ctx)
				if err != nil {
					logs.Logger.Warnf("s.client.Revoke(ctx, %x), %s.", s.leaseid, err)
				}
				goto start
			}
		}
	}
}

// Stop stop register
func (s *Agent) Stop() (err error) {
	ctx, _ := context.WithTimeout(context.TODO(), 3*time.Second)
	kv := clientv3.NewKV(s.client)
	_, err = kv.Delete(ctx, s.key)
	if err != nil {
		err = fmt.Errorf("Agent.Stop(), %s", err)
		return
	}
	s.ctxCancel()
	err = s.client.Close()
	return
}

func (s *Agent) keepAlive(ctx context.Context) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	value, _ := json.Marshal(s.value)
	resp, err := s.client.Grant(ctx, s.leaseTime)
	if err != nil {
		return nil, fmt.Errorf("s.client.Grant(ctx, %d), %s", s.leaseTime, err)
	}

	logs.Logger.Debugf("s.client.Grant(ctx, %d), leaseid: [%x].", s.leaseTime, resp.ID)

	_, err = s.client.Put(ctx, s.key, string(value), clientv3.WithLease(resp.ID))
	if err != nil {
		return nil, fmt.Errorf("s.client.Put(ctx, key, value, %x), %s", resp.ID, err)
	}

	s.leaseid = resp.ID
	logs.Logger.Debugf("etcd keepAlive ok, key: %s, leaseid: [%x].", s.key, s.leaseid)

	return s.client.KeepAlive(ctx, resp.ID)
}

// revoke .
func (s *Agent) revoke(ctx context.Context) error {
	_, err := s.client.Revoke(ctx, s.leaseid)
	if err != nil {
		return err
	}
	return s.client.Close()
}

// GetValues .
func (s *Agent) GetValues(ctx context.Context, path string) (kvs []KeyValue, err error) {
	if path == "" {
		err = fmt.Errorf("not set path")
		return
	}

	kv := clientv3.NewKV(s.client)
	var resp *clientv3.GetResponse

	if path[len(path)-1:] == "/" {
		resp, err = kv.Get(ctx, path, clientv3.WithPrefix())
	} else {
		resp, err = kv.Get(ctx, path)
	}

	if err != nil {
		return
	}

	for _, v := range resp.Kvs {
		kvs = append(kvs, KeyValue{string(v.Key), string(v.Value)})
	}
	return
}

// Close close agent
func (s *Agent) Close() error {
	return s.client.Close()
}

// Watcher .
type Watcher struct {
	ctx       context.Context
	ctxCancel context.CancelFunc

	Path   string
	Nodes  *cache.Cache
	client *clientv3.Client
	wcb    WatcherCB
}

// NewWatcher create watcher
func NewWatcher(path string, endpoints []string, wcb WatcherCB) (w *Watcher, err error) {
	// {[127.0.0.1:2379] 0s 2s 2s 6s 0 0 <nil>   false [] <nil> <nil>}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 3 * time.Second,
	})

	if err != nil {
		return nil, err
	}

	w = &Watcher{
		Path:   path,
		Nodes:  cache.NewCache(2000),
		client: cli,
		wcb:    wcb,
	}
	w.ctx, w.ctxCancel = context.WithCancel(context.Background())

	//go w.Start()
	return w, err
}

// Start start watch
func (w *Watcher) Start() {
	// opts := []clientv3.OpOption{clientv3.WithRev(0)}
	rch := clientv3.NewWatcher(w.client).Watch(context.Background(), w.Path, clientv3.WithPrefix())

	for {
		select {
		case <-w.ctx.Done():
			logs.Logger.Warnf("user cancel Watcher.")
			return
		case wresp := <-rch:
			for _, ev := range wresp.Events {
				switch ev.Type {
				case clientv3.EventTypePut:
					w.Nodes.AddORUpdate(string(ev.Kv.Key), ev.Kv.Value)
					if w.wcb != nil {
						w.wcb("PUT", string(ev.Kv.Key), string(ev.Kv.Value))
					}
				case clientv3.EventTypeDelete:
					w.Nodes.Delete(string(ev.Kv.Key))
					if w.wcb != nil {
						w.wcb("DELETE", string(ev.Kv.Key), string(ev.Kv.Value))
					}
				}
			}
		}
	}
}

// Stop stop watch
func (w *Watcher) Stop() {
	w.ctxCancel()
	w.client.Close()
}

// GetValues .
func (w *Watcher) GetValues(ctx context.Context, path string) (kvs []KeyValue, err error) {
	if path == "" {
		err = fmt.Errorf("not set path")
		return
	}

	kv := clientv3.NewKV(w.client)
	var resp *clientv3.GetResponse

	if path[len(path)-1:] == "/" {
		resp, err = kv.Get(ctx, path, clientv3.WithPrefix())
	} else {
		resp, err = kv.Get(ctx, path)
	}
	if err != nil {
		return
	}

	for _, v := range resp.Kvs {
		kvs = append(kvs, KeyValue{string(v.Key), string(v.Value)})
	}
	return
}

// Close close watcher
func (w *Watcher) Close() error {
	return w.client.Close()
}
