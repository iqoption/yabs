package main

import (
	"os"
	"fmt"
	"gopkg.in/urfave/cli.v2"
	"context"
	"gopkg.in/olivere/elastic.v5"
	"reflect"
	log "github.com/sirupsen/logrus"
)

const (
	AGE = `older`
	NAME = `name`
	SIZE = `count`
	SHOW = `show_only`
)

type Symbol struct {
	DirPath   string `json:"path"`
	Version   string `json:"build"`
	DebugId   string `json:"debugId"`
	Platform  string `json:"platform"`
	DataAdded string `json:"date_added"`
}

type Callback func(c *cli.Context, args cli.Args) error

var rmCallbacks = map[string]Callback{
	"symbols": rmSymbols,
}

func RemoveCommand() cli.Command {
	return cli.Command{
		Name:    "remove",
		Aliases: []string{"rm"},
		Action:  remove,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  AGE,
				Value: "16d",
			},
			cli.StringFlag{
				Name:  NAME,
				Value: ".*autotests", //Regular expression
			},
			cli.StringFlag{
				Name:URL,
				Value:"http://127.0.0.1:9200",
			},
			cli.IntFlag{
				Name:SIZE,
				Value: 1000,
			},
			cli.BoolFlag{
				Name: SHOW,
			},
		},
	}
}

func remove(c *cli.Context) error {
	initElasticClient(c.String(URL))

	if c.NArg() == 0 {
		message := `Empty task, available values:
	symbols
	crashes`
		fmt.Println(message)
		return fmt.Errorf("Empty task")
	}

	task := c.Args().Get(0)

	if cb, ok := rmCallbacks[task]; ok {
		return cb(c, c.Args().Tail())
	} else {
		fmt.Printf("Unknown task %s\n", task)
		return fmt.Errorf("Unknown task %s", task)
	}

	return nil
}

func rmSymbols(c *cli.Context, args cli.Args) error {

	older := c.String(AGE)
	name := c.String(NAME)
	size := c.Int(SIZE)
	showOnly := c.Bool(SHOW)

	rng := elastic.NewRangeQuery("date_added")
	rng.Lte(fmt.Sprintf("now-%s", older))


	query := elastic.NewBoolQuery().Must(rng, elastic.NewRegexpQuery("build", name))

	searchResult, err  := ElasticClient.Search().
		Index("breakpad").
		Type("symbol").
		Query(query).
		Sort("date_added", true).
		Size(size).
		Do(context.Background())
	if err != nil {
		log.WithError(err).Panic("Can't call to Elastic")
	}

	var styp Symbol
	for _, item := range searchResult.Each(reflect.TypeOf(styp)) {
		s := item.(Symbol)

		if showOnly {
			log.WithFields(log.Fields{
				"platform": s.Platform,
				"version": s.Version,
				"path": s.DirPath,
				"date": s.DataAdded,
			}).Info("Symbols")
			continue
		}

		deleter := elastic.NewDeleteService(ElasticClient).
			Index("breakpad").
			Type("symbol").
			Id(s.DebugId)

		_, err := deleter.Do(context.Background())
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"id": s.DebugId,
			}).Error("Can't remove document in Elastic")

			return err
		}

		err = os.RemoveAll(s.DirPath)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"path": s.DirPath,
			}).Error("Can't remove directory")

			return err
		} else {
			log.WithFields(log.Fields{
				"platform": s.Platform,
				"version": s.Version,
				"path": s.DirPath,
			}).Info("Removed symbols")
		}
	}

	return nil
}

