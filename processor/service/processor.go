package service

import (
	"errors"
	"os"
	"os/signal"
	"syscall"
	"github.com/streadway/amqp"
	"yabs/common/task"
	"yabs/common/data/base"
	"yabs/processor/cfg"
	"fmt"
	"yabs/common/format/minidump"
	"encoding/json"
	log "github.com/sirupsen/logrus"
)

const (
	SIGHUP            = syscall.SIGHUP
	SIGTERM           = syscall.SIGTERM
	DEVELOPER_VERSION = "999.999.999"
)

type RabbitClient struct {
	connection  *amqp.Connection
	taskChannel *amqp.Channel
	taskQueue   amqp.Queue
	messages    <-chan amqp.Delivery
	postChannel *amqp.Channel
}

type ProcessorService struct {
	SymbolsProcessor
	MinidumpProcessor
	WebdumpProcessor
	config     cfg.Config
	rabbit     *RabbitClient
	sig        <-chan os.Signal
	repository *base.Repository
}

type ReportWithId struct {
	minidump.Report
	Id string `json:"id"`
}

func newRabbitClient(conf cfg.Config) *RabbitClient {
	conn, err := amqp.Dial(conf.RabbitServer())
	failOnError(err, "Failed to connect to RabbitMQ")

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a taskChannel")

	q, err := ch.QueueDeclare(
		conf.RabbitQueue(),
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a taskQueue")

	err = ch.Qos(
		1,
		0,
		false,
	)
	failOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name, // taskQueue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	return &RabbitClient{connection: conn,
		taskChannel:                 ch,
		taskQueue:                   q,
		messages:                    msgs,
	}
}

func (p *ProcessorService) Init(config cfg.Config) error {
	p.config = config

	rabbit := newRabbitClient(p.config)
	if rabbit == nil {
		return errors.New("Can't connect to rabbit")
	}

	p.rabbit = rabbit
	p.createPostProcessingExchange()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, SIGHUP, SIGTERM)
	p.sig = sig

	os.MkdirAll(p.config.SymbolsPath(), 0777)

	var cache base.Cashe
	if len(p.config.Memcache()) > 0 {
		cache, _ = base.NewMemcache(p.config.Memcache())
	} else {
		cache, _ = base.NewRedis(p.config.RedisAddres(),
			p.config.RedisPassword())
	}

	rep, err := base.NewRepository(p.config.ElasticUrl(), cache)
	if err != nil {
		log.WithError(err).Error("Can't create repository")
		return err
	}
	p.repository = rep

	p.initSymbolProcessor(p.config.SymbolsPath(),
		p.repository)
	p.initMinidumpProcessor(p.config,
		p.repository)
	p.initWebdumpProcessor(p.config,
		p.repository)

	return nil
}

func (p *ProcessorService) Loop() {
	for {
		select {
		case msg := <-p.rabbit.messages:
			if p.handleTask(msg.Body) == nil {
				msg.Ack(false)
			} else {
				msg.Nack(false, true)
			}
		case sig := <-p.sig:
			p.handleSignal(sig)
		}
	}
}

func (p *ProcessorService) handleTask(message []byte) error {

	t := task.FromJson(message)
	if t == nil {
		return errors.New("Invalid task")
	}
	switch t.(type) {
	case *task.Dump:
		d, _ := t.(*task.Dump)
		r := p.handleMiniDump(d)
		p.sendNext(r)
	case *task.Symbol:
		s, _ := t.(*task.Symbol)
		p.handleSymbol(s)
	case *task.WebDump:
		w, _ := t.(*task.WebDump)
		r := p.handleWebDump(w)
		p.sendNext(r)
	}

	return nil
}

func (p *ProcessorService) handleSignal(sig os.Signal) {
	log.WithField("signal", sig.String()).
		Info("Catch")

	if sig == SIGHUP {
		p.reloadConfiguration()
	}
}

func (p *ProcessorService) createPostProcessingExchange() {
	if len(p.config.RabbitPostExchange()) == 0 {
		p.rabbit.postChannel = nil
		return
	}

	ch, err := p.rabbit.connection.Channel()
	failOnError(err, "Failed to open a taskChannel")
	err = ch.ExchangeDeclare(
		p.config.RabbitPostExchange(),
		p.config.RabbitPostType(),
		true,
		true,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare an exchange")
	p.rabbit.postChannel = ch
}

func (p *ProcessorService) sendNext(report *ReportWithId) {
	if p.rabbit.postChannel == nil {
		return
	}

	data, err := json.Marshal(report)
	if err != nil {
		log.WithError(err).
			Error("Can't serialize report with id")
	}

	err = p.rabbit.postChannel.Publish(
		p.config.RabbitPostExchange(),
		"",
		false,
		false,
		amqp.Publishing{
			ContentType: "text/json",
			Body:        data,
		})
	if err != nil {
		log.WithError(err).
			Error("Can't send crash to next stage")
	}
}

func (p *ProcessorService) reloadConfiguration() {
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

		if conf.LogLevel() != p.config.LogLevel() {
			err := p.changeLevel(conf.LogLevel())
			if err != nil {
				noErrors = false
			}
		}

		if len(conf.WebBlackListSignaturs()) != len(p.config.WebBlackListSignaturs()) {
			p.initWebdumpProcessor(conf, p.repository)
		}

		if noErrors {
			cfg.GlobalConfig = conf
			p.config = conf
			log.Info("Reloaded configuration")
		}
	}
}

func (p *ProcessorService) changeLevel(l string) error {
	level, err := log.ParseLevel(l)
	if err != nil {
		log.WithError(err).
			Warn("Can't parse level")
		return err
	}

	log.WithFields(log.Fields{
		"old level": p.config.LogLevel(),
		"new level": l,
	}).
		Info("Change log level")
	log.SetLevel(level)
	return nil
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}
