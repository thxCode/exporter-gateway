package transfer

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/juju/errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/rancher/exporter-gateway/pkg/exporter"
	"github.com/rancher/exporter-gateway/pkg/flag"
	"github.com/rancher/exporter-gateway/pkg/pushgateway"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func Run(cliCtx *cli.Context) error {
	transfer, err := createTransfer(cliCtx)
	if err != nil {
		return errors.Annotate(err, "failed to create the gateway")
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	transfer.run(sigCh)
	transfer.clean()

	return nil
}

type metrics struct {
	scraper string
	mfs     map[string]*dto.MetricFamily
}

type transfer struct {
	timeout      time.Duration
	interval     time.Duration
	grouping     map[string]string
	sources      []*exporter.Endpoint
	destinations []*pushgateway.Endpoint
}

func (t *transfer) scrape(ctx context.Context, mfChan chan<- metrics) {
	for _, src := range t.sources {
		scraperName := src.GetName()

		ret, err := src.Scrape(ctx, t.timeout)
		if err != nil {
			log.WithError(err).Warnf("'%s' scrape failure", scraperName)
			continue
		}

		mfChan <- metrics{scraper: scraperName, mfs: ret}
		log.Debugf("'%s' scrape success", scraperName)
	}
}

func (t *transfer) push(ctx context.Context, mfChan <-chan metrics) {
	for m := range mfChan {
		for _, dest := range t.destinations {
			scraperName, pusherName := m.scraper, dest.GetName()

			err := dest.Put(ctx, scraperName, t.grouping, m.mfs)
			if err != nil {
				log.WithError(err).Warnf("'%s' push '%s' failure", pusherName, scraperName)
				continue
			}

			log.Debugf("'%s' push '%s' success", pusherName, scraperName)
		}
	}
}

func (t *transfer) run(sigCh <-chan os.Signal) {
	mfChan := make(chan metrics, len(t.sources))
	runCtx, cancelRunFn := context.WithCancel(context.Background())
	defer cancelRunFn()

	// scrape interval loop
	go func(runCtx context.Context) {
		var intervalTimer *time.Timer

		for {
			select {
			case <-sigCh:
				close(mfChan)
				return
			default:
			}

			scrapeCtx, cancelScrapeFn := context.WithCancel(runCtx)
			go t.scrape(scrapeCtx, mfChan)

			if intervalTimer == nil {
				intervalTimer = time.NewTimer(t.interval)
			} else {
				intervalTimer.Reset(t.interval)
			}

			select {
			case <-sigCh:
				cancelScrapeFn()
				close(mfChan)
				return
			case <-intervalTimer.C:
				cancelScrapeFn()
			}
		}
	}(runCtx)

	t.push(runCtx, mfChan)
}

// clean should finish the deleting in 15s
func (t *transfer) clean() {
	wg := &sync.WaitGroup{}
	cleanCtx, cancelCleanFn := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelCleanFn()

	// delete pushgateway metrics
	for _, src := range t.sources {
		for _, dst := range t.destinations {
			wg.Add(1)

			go func(ctx context.Context, src *exporter.Endpoint, dst *pushgateway.Endpoint) {
				defer wg.Done()
				scraperName, pusherName := src.GetName(), dst.GetName()

				err := dst.Delete(ctx, scraperName, t.grouping)
				if err != nil {
					log.WithError(err).Warnf("'%s' delete '%s' failure", pusherName, scraperName)
					return
				}

				log.Debugf("'%s' delete '%s' success", pusherName, scraperName)
			}(cleanCtx, src, dst)
		}
	}

	wg.Wait()
}

func createTransfer(cliCtx *cli.Context) (*transfer, error) {
	var (
		interval    = cliCtx.Duration("interval")
		fromFlag    = cliCtx.Generic("from").(*flag.Routers)
		fromTimeout = cliCtx.Duration("from.timeout")
		toFlag      = cliCtx.Generic("to").(*flag.Routers)
		toGroupFlag = cliCtx.Generic("to.group").(*flag.Labels)
	)

	if fromTimeout > interval {
		return nil, errors.New("timeout must less than interval")
	}
	log.Infof("transferring in %v per %v", fromTimeout, interval)

	from := fromFlag.Unwrap()
	if len(from) == 0 {
		return nil, errors.New("cannot start without any exporter endpoints")
	}
	log.Infof("transferring from %s", fromFlag)

	to := toFlag.Unwrap()
	if len(to) == 0 {
		return nil, errors.New("cannot start without any pushgateway endpoints")
	}
	log.Infof("transferring to %s", toFlag)

	grouping := toGroupFlag.Unwrap()
	log.Infof("transferring with labels %s", toGroupFlag)

	var (
		sources      []*exporter.Endpoint
		destinations []*pushgateway.Endpoint
	)
	// parse sources
	for _, f := range from {
		src, err := exporter.NewEndpoint(f)
		if err != nil {
			return nil, errors.Annotatef(err, "failed to create %s exporter endpoint", f.Name)
		}

		sources = append(sources, src)
	}
	// parse destinations
	for _, t := range to {
		dest, err := pushgateway.NewEndpoint(t)
		if err != nil {
			return nil, errors.Annotatef(err, "failed to create %s pushgateway endpoint", t.Name)
		}

		destinations = append(destinations, dest)
	}

	return &transfer{
		fromTimeout,
		interval,
		grouping,
		sources,
		destinations,
	}, nil
}
