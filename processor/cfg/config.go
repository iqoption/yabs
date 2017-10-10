package cfg

import (
	"os"
	"errors"
	"encoding/json"
	log "github.com/sirupsen/logrus"
)

type Config interface {
	SymbolsPath() string
	RabbitServer() string
	RabbitQueue() string
	RabbitPostExchange() string
	RabbitPostType() string
	ElasticUrl() string
	Memcache() []string
	RedisAddres() string
	RedisPassword() string
	LogLevel() string
	WebBlackListSignaturs() []string
}

var GlobalConfig Config
var GlobalConfigPath string

func FromJson(pathTo string) (Config, error) {
	file, err := os.Open(pathTo)
	if err != nil {
		log.WithError(err).Error("Get config failed")
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var jconf JsonConfig
	err = decoder.Decode(&jconf)
	if err != nil {
		log.WithError(err).Error("Error at cfg parsing")
		return nil, err
	}

	if len(jconf.SymbolsPathName) == 0 {
		return nil, errors.New("symbols_pathname can't is empty")
	}

	return &jconf, nil
}