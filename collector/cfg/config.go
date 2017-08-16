package cfg

import (
	"os"
	"errors"
	"encoding/json"
	"sync"
	log "github.com/sirupsen/logrus"
)

type Config interface {
	Port() uint
	Host() string
	SymbolsTmpDir() string
	DumpsTmpDir() string
	RabbitServer() string
	RabbitQueue() string
	LogLevel() string

	// monitoring
	MonitoringEnable() bool
	FlushTimeout() int
	FlushBufferSize() int
	UdpAddress() string
}

var GlobalConfigMutex sync.Mutex
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

	if len(jconf.TemproryDirs.Symbols) == 0 {
		return nil, errors.New("The path to the temporary symbol directory is not set")
	}

	if len(jconf.TemproryDirs.Dumps) == 0 {
		return nil, errors.New("The path to the temporary dump directory is not set")
	}

	return &jconf, nil
}
