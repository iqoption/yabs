package cfg

type RabbitCfg struct {
	Server   string `json:"server"`
	Queue    string `json:"queue"`
	Exchange string `json:"post-exchange"`
	Type     string `json:"post-type"`
}

type RedisCfg struct {
	Address  string `json:"address"`
	Password string `json:"password"`
}

type CacheCfg struct {
	Memcached []string `json:"memcache"`
	Redis     RedisCfg `json:"redis"`
}

type LogCfg struct {
	Level string `json:"level"`
}

type JsonConfig struct {
	SymbolsPathName   string     `json:"symbols_pathname"`
	Rabbit            *RabbitCfg `json:"rabbit_cfg"`
	Cache             *CacheCfg  `json:"cache"`
	Elastic           string     `json:"elastic"`
	Log               *LogCfg    `json:"log"`
	WebBListSignaturs []string   `json:"web_blacklist_signaturs"`
}

func (cfg *JsonConfig) SymbolsPath() string {
	return cfg.SymbolsPathName
}

func (cfg *JsonConfig) RabbitServer() string {
	return cfg.Rabbit.Server
}

func (cfg *JsonConfig) RabbitQueue() string {
	return cfg.Rabbit.Queue
}

func (cfg *JsonConfig) ElasticUrl() string {
	return cfg.Elastic
}

func (cfg *JsonConfig) Memcache() []string {
	return cfg.Cache.Memcached
}

func (cfg *JsonConfig) RedisAddres() string {
	return cfg.Cache.Redis.Address
}

func (cfg *JsonConfig) RedisPassword() string {
	return cfg.Cache.Redis.Password
}

func (cfg *JsonConfig) WebBlackListSignaturs() []string {
	return cfg.WebBListSignaturs;
}

func (cfg *JsonConfig) RabbitPostExchange() string {
	return cfg.Rabbit.Exchange
}

func (cfg *JsonConfig) RabbitPostType() string {
	return cfg.Rabbit.Type
}

func (cfg *JsonConfig) LogLevel() string {
	return cfg.Log.Level
}
