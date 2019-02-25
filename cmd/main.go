package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/rancher/exporter-gateway/pkg/flag"
	"github.com/rancher/exporter-gateway/pkg/transfer"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	VER  = "dev"
	HASH = "-"
)

func main() {
	app := cli.NewApp()
	app.Version = fmt.Sprintf("%s(%s)", VER, HASH)
	app.Name = "exporter-gateway"
	app.Usage = "Gateway for the Prometheus exporters"
	app.Flags = []cli.Flag{
		cli.DurationFlag{
			Name:  "interval",
			Usage: "[optional] Set the transmission interval",
			Value: 15 * time.Second,
		},
		cli.GenericFlag{
			Name:  "from",
			Usage: "Set the exporter routers, for example, transferring from localhost xyz exporter by '--from xyz-exporter.url=https://127.0.0.1:9010/metrics'",
			Value: new(flag.Routers),
		},
		cli.DurationFlag{
			Name:  "from.timeout",
			Usage: "[optional] Set timeout for scraping, must less than --interval",
			Value: 10 * time.Second,
		},
		cli.GenericFlag{
			Name:  "to",
			Usage: "Set the pushgateway routers, for example, transferring to x.y.z pushgateway by '--to xyz-pushgateway.url=http://x.y.z:9091/metrics'",
			Value: new(flag.Routers),
		},
		cli.GenericFlag{
			Name:  "to.group",
			Usage: "[optional] Set group for the metrics, for example, grouping all metrics by '--to.group instance=l.m.n'",
			Value: new(flag.Labels),
		},
		cli.BoolFlag{
			Name:  "log.json",
			Usage: "[optional] Log as JSON",
		},
		cli.BoolFlag{
			Name:  "log.debug",
			Usage: "[optional] Log debug info",
		},
	}
	app.Before = func(context *cli.Context) error {
		if context.Bool("log.json") {
			log.SetFormatter(&log.JSONFormatter{})
		} else {
			log.SetFormatter(&log.TextFormatter{ForceColors: true, FullTimestamp: true})
		}

		if context.Bool("log.debug") {
			log.SetLevel(log.DebugLevel)
			runtime.SetBlockProfileRate(20)
			runtime.SetMutexProfileFraction(20)
		}

		log.SetOutput(os.Stdout)

		return nil
	}
	app.Action = transfer.Run

	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}
