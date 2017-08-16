package main

import (
	"flag"
	"fmt"
	"yabs/collector/cfg"
	"yabs/collector/api"
	"os"
	"os/signal"
	log "github.com/sirupsen/logrus"
	"syscall"
)

var Build string
var Version string

const (
	SIGHUP = syscall.SIGHUP
)

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
			log.WithError(err).Fatal("Error reading configuration file")
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
}

func HandleError(err error) {
	if err != nil {
		panic(fmt.Sprintf("Error: %s", err.Error()))
	}
}

func main() {
	go func() {
		var service api.GinCollectorService
		HandleError(service.Init())
		HandleError(service.Start())
	}()

	signals := make(chan os.Signal)
	signal.Notify(signals, SIGHUP)
	for {
		select {
		case sig := <-signals:
			handleSignal(sig)
		}
	}

}

func handleSignal(sig os.Signal) {
	if sig != SIGHUP {
		return
	}
	cfg.GlobalConfigMutex.Lock()
	defer cfg.GlobalConfigMutex.Unlock()

	log.Info("Try to reload configuration")
	if len(cfg.GlobalConfigPath) != 0 {
		conf, err := cfg.FromJson(cfg.GlobalConfigPath)
		if err != nil {
			log.WithError(err).
				Error("Error reading configuration file")
			return
		}
		//TODO need compare all configuration and apply only a changes
		noErrors := true

		if conf.LogLevel() != cfg.GlobalConfig.LogLevel() {
			err := changeLevel(conf.LogLevel())
			if err != nil {
				noErrors = false
			}
		}

		if noErrors {
			cfg.GlobalConfig = conf
			log.Info("Reloaded configuration")
		}
	}
}

func changeLevel(l string) error {
	level, err := log.ParseLevel(l)
	if err != nil {
		log.WithError(err).
			Warn("Can't parse level")
		return err
	}

	log.WithFields(log.Fields{
		"old level": cfg.GlobalConfig.LogLevel(),
		"new level": l,
	}).
		Info("Change log level")
	log.SetLevel(level)
	return nil
}