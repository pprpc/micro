package svc

// 服务配置

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"xcthings.com/hjyz/common"
	"xcthings.com/hjyz/logs"
)

// Config micro config get
type Config struct {
	agent   *Agent
	region  string
	lanip   string
	name    string
	dbs     []string
	private bool
	Conf    *MSConfig
	lisURI  []string
}

// NewConfig create config
func NewConfig(a *Agent, region, lanip, microName string, dbs []string, private bool) (cfg *Config, err error) {
	if a == nil {
		err = fmt.Errorf("Agent not init")
		return
	}
	if region == "" || microName == "" || lanip == "" {
		err = fmt.Errorf("region/microName/lanip must be set")
		return
	}
	cfg = new(Config)
	cfg.agent = a
	cfg.region = region
	cfg.lanip = lanip
	cfg.name = microName
	cfg.private = private
	cfg.dbs = dbs
	cfg.Conf = new(MSConfig)

	return
}

// GetAll return MSConfig
func (c *Config) GetAll() (err error) {
	err = c.PublicConf()
	if err != nil {
		return
	}
	err = c.ListenConf()
	if err != nil {
		return
	}

	err = c.LogConf()
	if err != nil {
		return
	}

	err = c.DBConfig()
	if err != nil {
		return
	}

	err = c.PpmqcliConf()
	if err != nil {
		logs.Logger.Warnf("c.PpmqcliConf(), %s.", err)
	}
	if c.private {
		err = c.PrivateConf()
		if err != nil {
			return
		}
	}
	return
}

// PublicConf get public config
// key: /conf/region/lanip/msname/public
func (c *Config) PublicConf() (err error) {
	key := fmt.Sprintf("/conf/%s/%s/%s/public", c.region, c.lanip, c.name)
	var _t PublicConf
	err = c.getValueObj(key, &_t)
	if err != nil {
		return
	}
	_t.ServerID = c.lanip
	c.Conf.Public = _t
	return
}

// ListenConf get listen config
// key: /conf/region/listen/lanip/msname
func (c *Config) ListenConf() (err error) {
	key := fmt.Sprintf("/conf/%s/%s/%s/listen", c.region, c.lanip, c.name)
	var _t []LisConf
	err = c.getValueObj(key, &_t)
	if err != nil {
		return
	}
	for _, row := range _t {
		c.lisURI = append(c.lisURI, row.URI)
	}
	c.Conf.Listen = _t
	return
}

// LogConf get log config
// key: /conf/region/log/lanip/msname
func (c *Config) LogConf() (err error) {
	key := fmt.Sprintf("/conf/%s/%s/%s/log", c.region, c.lanip, c.name)
	var _t ValueLogConf
	err = c.getValueObj(key, &_t)
	if err != nil {
		return
	}
	c.Conf.Log = _t
	return
}

// DBConfig get db config
// key: /conf/region/db/dbname
func (c *Config) DBConfig() (err error) {
	var dbs []ValueDbconf
	for _, row := range c.dbs {
		key := fmt.Sprintf("/conf/%s/db/%s", c.region, row)
		var _t ValueDbconf
		err = c.getValueObj(key, &_t)
		if err != nil {
			return
		}
		dbs = append(dbs, _t)
	}
	c.Conf.Dbs = dbs
	return
}

// PpmqcliConf get ppmqcli config
// key: /conf/region/ppmqcli/lanip/msname
func (c *Config) PpmqcliConf() (err error) {
	key := fmt.Sprintf("/conf/%s/%s/%s/ppmqcli", c.region, c.lanip, c.name)
	var _t, _m []PpmqcliConf
	err = c.getValueObj(key, &_t)
	if err != nil {
		return
	}

	for _, v := range _t {
		if v.Class == "localmqd" {
			// localmqd register,get tcp listen
			// replace URL
			// key: /register/region/localmqd/lanip
			v.HWFeature = fmt.Sprintf("localmqd-%s-%s-%s", c.region, c.lanip, c.name)

			key = fmt.Sprintf("/register/%s/localmqd/", c.region)
			var _mqs ValueRegService
			err = c.getValueObj(key, &_mqs)
			if err != nil {
				err = fmt.Errorf("getValueObj(%s), %s", key, err)
				return
			}
			//for _, reg := range _mqs {
			v.URL, err = GetTCPURL(_mqs)
			if err != nil {
				err = fmt.Errorf("GetTCPURL: %v, %s", _mqs, err)
				return
			}
			//}
		} else if v.Class == "ppmqd" {
			// ppmqd register, get tcp listen
			// replace URL
			// key: /register/region/localmqd/lanip
			v.HWFeature = fmt.Sprintf("ppmqd-%s-%s-%s", c.region, c.lanip, c.name)

			key = fmt.Sprintf("/register/%s/ppmqd/", c.region)
			var _mqs ValueRegService
			err = c.getValueObj(key, &_mqs)
			if err != nil {
				err = fmt.Errorf("getValueObj(%s), %s", key, err)
				return
			}
			v.URL, err = GetTCPURL(_mqs)
			if err != nil {
				err = fmt.Errorf("GetTCPURL: %v, %s", _mqs, err)
			}
		}
		_m = append(_m, v)
	}

	c.Conf.Ppmqclis = _m
	return
}

// PrivateConf get private config
// key: /conf/region/private/lanip/msname
func (c *Config) PrivateConf() (err error) {
	key := fmt.Sprintf("/conf/%s/%s/%s/private", c.region, c.lanip, c.name)
	var _t json.RawMessage
	err = c.getValueObj(key, &_t)
	if err != nil {
		return
	}
	c.Conf.PrivateConfig = _t
	return
}

func (c *Config) getValueObj(key string, obj interface{}) (err error) {
	var kvs []KeyValue
	ctx, _ := context.WithTimeout(context.TODO(), 3*time.Second)

	kvs, err = c.agent.GetValues(ctx, key)
	if err != nil {
		return
	}
	if len(kvs) == 0 {
		err = fmt.Errorf("get key: %s, value is null", key)
		return
	}

	err = json.Unmarshal([]byte(kvs[0].Value), obj)
	if err != nil {
		err = fmt.Errorf("json.Unmarshal(%s), %s", kvs[0].Value, err)
		return
	}
	return
}

// GetTCPURL get listen tcp url
func GetTCPURL(reg ValueRegService) (url string, err error) {
	if reg.LanIP == "" || len(reg.Listen) == 0 {
		err = fmt.Errorf("ValueRegService value: LanIP/Listen is error")
		return
	}
	var port int32
	for _, lis := range reg.Listen {
		port, err = getTCPPort(lis)
		if err != nil {
			continue
		}
		break
	}
	if port == 0 {
		err = fmt.Errorf("not find listen: [%v] tcp uri", reg.Listen)
		return
	}

	url = fmt.Sprintf("tcp://%s:%d", reg.LanIP, port)
	return
}

func getTCPPort(lis LisConf) (port int32, err error) {
	u, e := url.ParseRequestURI(lis.URI)
	if e != nil {
		logs.Logger.Warnf("url.ParseRequestURI(%s), %s.", lis, e)
		return
	}
	if u.Scheme != "tcp" {
		err = fmt.Errorf("uri scheme : %s, not tcp", u.Scheme)
		return
	}
	_t, e := strconv.Atoi(u.Port())
	if e == nil {
		port = int32(_t)
		return
	}

	err = e
	return
}

func getUDPPort(lis LisConf) (port int32, err error) {
	u, e := url.ParseRequestURI(lis.URI)
	if e != nil {
		logs.Logger.Warnf("url.ParseRequestURI(%s), %s.", lis, e)
		return
	}
	if u.Scheme != "udp" {
		err = fmt.Errorf("uri scheme : %s, not udp", u.Scheme)
		return
	}
	_t, e := strconv.Atoi(u.Port())
	if e == nil {
		port = int32(_t)
		return
	}

	err = e
	return
}

// GetListenURI  get listen uri
func (c *Config) GetListenURI() []string {
	return c.lisURI
}

//GetWanIP get wan ipaddr.
func (c *Config) GetWanIP(lanip string) (wanip string, err error) {
	key := fmt.Sprintf("/conf/%s/getwanip/%s", c.region, lanip)

	var kvs []KeyValue
	ctx, _ := context.WithTimeout(context.TODO(), 3*time.Second)
	kvs, err = c.agent.GetValues(ctx, key)
	if err != nil {
		return
	}
	if len(kvs) == 0 {
		err = fmt.Errorf("get key: %s, value is null", key)
		return
	}
	wanip = string(kvs[0].Value)
	return
}

// GetListenTCPPorts get listen tcp ports
func GetListenTCPPorts(liss []LisConf) (ports []int32) {
	for _, row := range liss {
		_t, err := getTCPPort(row)
		if err != nil {
			continue
		}
		ports = append(ports, _t)
	}
	return
}

// GetListenUDPPorts get listen udp ports
func GetListenUDPPorts(liss []LisConf) (ports []int32) {
	for _, row := range liss {
		_t, err := getUDPPort(row)
		if err != nil {
			continue
		}
		ports = append(ports, _t)
	}
	return
}

// GetListenURI get  listen uri
func GetListenURI(liss []LisConf) (uris []string) {
	for _, row := range liss {
		uris = append(uris, row.URI)
	}
	return
}

// GetListenResID get listen resid
func GetListenResID(liss []LisConf) (resid []int) {
	for _, row := range liss {
		resid = append(resid, row.ResID)
	}

	resid = common.RemoveDuplicatesInt(resid)
	return
}
