package svc

import "encoding/json"

const (
	RES_PPMQD_TCP       int32 = 1
	RES_PPMQD_UDP       int32 = 2
	RES_PPMQD_MQTT      int32 = 3
	RES_APIGW_GRPC      int32 = 4
	RES_APIGW_HTTP      int32 = 5
	RES_APIGW_PPRPC     int32 = 6
	RES_FTCONN_NAT      int32 = 8
	RES_FTCONN_RELAY    int32 = 9
	RES_FTCONN_P2P      int32 = 10
	RES_FTCONN_LIVES    int32 = 11
	RES_GLBS_TCP        int32 = 13
	RES_GLBS_UDP        int32 = 14
	RES_APIGW_GRPC_TLS  int32 = 104
	RES_APIGW_HTTP_TLS  int32 = 105
	RES_APIGW_PPRPC_TLS int32 = 106
)

// KeyValue etcd key value pair
type KeyValue struct {
	Key   string
	Value string
}

// ValueWan lan wan  pair
// key: /conf/region/getwanip/lanip
type ValueWan struct {
	WanIP string `json:"wanip,omitempty"`
}

// ValueDbconf db conf
// Key: /conf/region/db/dbname
type ValueDbconf struct {
	ConfName string `json:"conf_name,omitempty"`
	Type     string `json:"type,omitempty"`
	User     string `json:"user,omitempty"`
	Pass     string `json:"pass,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Name     string `json:"name,omitempty"`
	Charset  string `json:"charset,omitempty"`
	Socket   string `json:"socket,omitempty"`
	MaxIdle  int    `json:"max_idle,omitempty"`
	MaxConn  int    `json:"max_conn,omitempty"`
	Debug    bool   `json:"debug,omitempty"`
}

// RedisConf Redis
type RedisConf struct {
	// host:port address.
	Addr     string `json:"addr,omitempty"`
	Password string `json:"password,omitempty"`
	DB       int    `json:"db,omitempty"`
	PoolSize int    `json:"pool_size,omitempty"`
	IdleConn int    `json:"idle_conn,omitempty"`
}

// ValueLogConf log conf
// key: /conf/region/log/lanip/msname
type ValueLogConf struct {
	File       string `json:"file,omitempty"`
	MaxSize    int    `json:"max_size,omitempty"`
	MaxBackups int    `json:"max_backups,omitempty"`
	MaxAge     int    `json:"max_age,omitempty"`
	Caller     bool   `json:"caller,omitempty"`
	Level      int8   //zapcore.Level `json:"level"`
	SeelogPort int    `json:"seelog_port,omitempty"`
	SeelogUser string `json:"seelog_user,omitempty"`
	SeelogPass string `json:"seelog_pass,omitempty"`
	LogDir     string `json:"log_dir,omitempty"`
}

// ValueRegService register service struct define
// key: /register/region/msname/lanip
type ValueRegService struct {
	Region string    `json:"region,omitempty"`
	Name   string    `json:"name,omitempty"`
	ResSrv []int     `json:"res_srv,omitempty"`
	LanIP  string    `json:"lan_ip,omitempty"`
	Listen []LisConf `json:"listen,omitempty"`
}

// LisConf listen conf
// key: /conf/region/listen/lanip/msname
type LisConf struct {
	URI         string `json:"uri,omitempty"` // tcp://ip:port, udp://ip:port
	ReadTimeout int64  `json:"read_timeout,omitempty"`
	TLSCrt      string `json:"tls_crt,omitempty"`
	TLSKey      string `json:"tls_key,omitempty"`
	ResID       int    `json:"res_id,omitempty"`
}

// PublicConf public config
// key: /conf/region/public/lanip/msname
type PublicConf struct {
	ReportInterval int64  `json:"report_interval,omitempty"`
	AdminProf      bool   `json:"admin_prof,omitempty"`
	AdminPort      int    `json:"admin_port,omitempty"`
	MaxGo          int    `json:"max_go,omitempty"`
	RunGo          bool   `json:"run_go,omitempty"`
	ServerID       string `json:"server_id,omitempty"`
}

// PpmqcliConf ppmq client config
// key: /conf/region/ppmqcli/lanip/msname
type PpmqcliConf struct {
	Class       string `json:"class,omitempty"` // localmqd ppmqd
	URL         string `json:"url,omitempty"`
	Account     string `json:"account,omitempty"`
	Password    string `json:"password,omitempty"`
	HWFeature   string `json:"hw_feature,omitempty"`
	TopicPrefix string `json:"topic_prefix,omitempty"`
	// 20180926
	MsgType  int32 `json:"msg_type,omitempty"`
	MsgCount int32 `json:"msg_count,omitempty"`
}

// MSConfig micro service config
type MSConfig struct {
	Public        PublicConf      `json:"public,omitempty"`         // key: /conf/region/public/lanip/msname
	Listen        []LisConf       `json:"listen,omitempty"`         // key: /conf/region/listen/lanip/msname
	Log           ValueLogConf    `json:"log,omitempty"`            // key: /conf/region/log/lanip/msname
	Dbs           []ValueDbconf   `json:"dbs,omitempty"`            // key: /conf/region/db/dbname
	Ppmqclis      []PpmqcliConf   `json:"ppmqclis,omitempty"`       // key: /conf/region/ppmqcli/lanip/msname
	PrivateConfig json.RawMessage `json:"private_config,omitempty"` // key: /conf/region/private/lanip/msname
}

// MicroClient micrl service client
type MicroClient struct {
	Name string   `json:"name,omitempty"`
	URIS []string `json:"uris,omitempty"`
}
