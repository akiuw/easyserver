package easyserver

// decode from config.json
type Options struct {
	Services []*ServiceOpt `json:"services,omitempty"`
	MQOption *NSQOpt       `json:"nsq,omitempty"`
}

type NSQOpt struct {
	Address    string `json:"address,omitempty"`
	LookupAddr string `json:"lookupaddr,omitempty"`
}

type ServiceOpt struct {
	Name       string `json:"name,omitempty"`
	ListenPort string `json:"listen_port,omitempty"`
	Redis      string `json:"redis,omitempty"`
	Database   string `json:"database,omitempty"`
	PoolConns  int    `json:"pool_conns,omitempty"`
}
