package main

import (
	"flag"
	"fmt"
	"yabs/processor/cfg"
	"yabs/processor/service"
	"os"
	log "github.com/sirupsen/logrus"
)

var Build string
var Version string

func init() {

	var cPath string
	var showVersion bool = false
	var showBuild bool = false

	flag.StringVar(&cPath, "config", "", "path to configuration file")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.BoolVar(&showBuild, "build", false, "show build")
	flag.Parse()

	if showVersion {
		fmt.Printf("Version: %s\n", Version)
		os.Exit(0)
	}

	if showBuild {
		fmt.Printf("Build: %s\n", Build)
		os.Exit(0)
	}

	if cPath != "" {
		conf, err := cfg.FromJson(cPath)
		if err != nil {
			log.WithError(err).
				Panic("Error reading configuration file")
		}

		cfg.GlobalConfig = conf
		cfg.GlobalConfigPath = cPath
	} else {
		flag.PrintDefaults()
		log.Fatal("Config file is not set")
	}

	level, err := log.ParseLevel(cfg.GlobalConfig.LogLevel())
	if err == nil {
		log.WithField("level", level).
			Info("Change log level")
		log.SetLevel(level)
	} else {
		log.WithError(err).Warning("Can't setup log level")
	}

	//todo need ping to Elastic
}

func HandleError(err error) {
	if err != nil {
		panic(fmt.Sprintf("Error: %s", err.Error()))
	}
}

func main() {
	processor := service.ProcessorService{}
	HandleError(processor.Init(cfg.GlobalConfig))
	processor.Loop()
}
