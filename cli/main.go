package main

import (
	"os"
	"gopkg.in/urfave/cli.v2"
	"gopkg.in/olivere/elastic.v5"
	log "github.com/sirupsen/logrus"

)

const (
	URL = `url`
)
var ElasticClient *elastic.Client = nil

func init()  {
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stdout)
}

func main() {
	app := cli.NewApp()
	app.Name = "crashes-cli: command line utils for YABS"

	app.Commands = []cli.Command{
		RemoveCommand(),
	}
	app.Run(os.Args)
}

func initElasticClient(url string) {
	c, err := elastic.NewClient(elastic.SetURL(url))

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"url": url,
		}).Fatal("Can't create ElasticSearch client")
	}
	ElasticClient = c
}