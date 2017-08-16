package service

import (
	"yabs/common/task"
	"encoding/json"
	"yabs/collector/cfg"
	"github.com/streadway/amqp"
	"github.com/go-errors/errors"
	logger "github.com/sirupsen/logrus"
)

type RabbitClient struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	queue      amqp.Queue
}

type CollectorService struct {
	cfg    cfg.Config
	rabbit *RabbitClient
}

func (s *CollectorService) AddSymbol(symbol string, description string) error {
	t := task.CreateSymbolTask(symbol, description)
	msg, err := json.Marshal(t)
	if err != nil {
		logger.WithError(err).Error("Can't serialize message")
		return err
	}
	return s.publish(msg)
}

func (s *CollectorService) AddSymbols(symbols []string, description string) error {
	t := task.CreateSymbolsTask(symbols, description)
	msg, err := json.Marshal(t)
	if err != nil {
		logger.WithError(err).Error("Can't serialize message")
		return err
	}

	return s.publish(msg)
}

func (s *CollectorService) AddMinidump(minidump, info, log string) error {
	t := task.CreateDumpTask(minidump, info, log)
	msg, err := json.Marshal(t)
	if err != nil {
		logger.WithError(err).Error("Can't serialize message")
		return err
	}
	return s.publish(msg)
}

func (s *CollectorService) AddWebDump(webdump string, info string) error {
	t := task.CreateWebDumpTask(webdump, info)
	msg, err := json.Marshal(t)
	if err != nil {
		logger.WithError(err).Error("Can't serialize message")
		return err
	}
	return s.publish(msg)
}

func (s *CollectorService) publish(msg []byte) error {
	return s.rabbit.channel.Publish("",
		s.rabbit.queue.Name,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         msg,
		})
}

func newRabbitClient(conf cfg.Config) *RabbitClient {
	conn, err := amqp.Dial(conf.RabbitServer())
	if err != nil {
		logger.WithError(err).Error("Failed to connect to RabbitMQ")
		return nil
	}

	ch, err := conn.Channel()
	if err != nil {
		logger.WithError(err).Error("Failed to open a channel")
	}

	q, err := ch.QueueDeclare(
		conf.RabbitQueue(),
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		logger.WithError(err).Error("Failed to declare a queue")
		return nil
	}

	return &RabbitClient{conn, ch, q}
}

func NewCollector(c cfg.Config) (*CollectorService, error) {
	client := newRabbitClient(c)
	if client == nil {
		logger.Error("Can't connect to rabbit")
		return nil, errors.New("Can't connect to rabbit")
	}

	return &CollectorService{c, client}, nil
}
