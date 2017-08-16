package cfg

type WebServerCfg struct {
	Port uint `json:"port"`
	Host string `json:"host"`
}

type TemproryDirs struct {
	Symbols string `json:"symbols"`
	Dumps   string `json:"dumps"`
}

type RabbitCfg struct {
	Server string `json:"server"`
	Queue  string `json:"queue"`
}

type LogCfg struct {
	Level string `json:"level"`
}

type MonitoringCfg struct {
	Enable          bool   `json:"enable"`
	FlushTimeout    int    `json:"flush_timeout"`
	FlushBufferSize int    `json:"flush_buffer_size"`
	UdpAddress      string `json:"udp_addr"`
}

type JsonConfig struct {
	TemproryDirs *TemproryDirs  `json:"temprory_dirs"`
	Server       *WebServerCfg  `json:"web_server"`
	Rabbit       *RabbitCfg     `json:"rabbit_cfg"`
	Log          *LogCfg        `json:"log"`
	Monitoring   *MonitoringCfg `json:"monitoring"`
}

func (cfg *JsonConfig) Port() uint {
	return cfg.Server.Port
}

func (cfg *JsonConfig) Host() string {
	return cfg.Server.Host
}

func (cfg *JsonConfig) SymbolsTmpDir() string {
	return cfg.TemproryDirs.Symbols
}

func (cfg *JsonConfig) DumpsTmpDir() string {
	return cfg.TemproryDirs.Dumps
}

func (cfg *JsonConfig) RabbitServer() string {
	return cfg.Rabbit.Server
}

func (cfg *JsonConfig) RabbitQueue() string {
	return cfg.Rabbit.Queue
}

func (cfg *JsonConfig) LogLevel() string {
	return cfg.Log.Level
}

func (cfg *JsonConfig) FlushTimeout() int {
	return cfg.Monitoring.FlushTimeout
}

func (cfg *JsonConfig) FlushBufferSize() int {
	return cfg.Monitoring.FlushBufferSize
}

func (cfg *JsonConfig) UdpAddress() string {
	return cfg.Monitoring.UdpAddress
}

func (cfg *JsonConfig) MonitoringEnable() bool {
	return cfg.Monitoring.Enable
}