package exporter

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/juju/errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/rancher/exporter-gateway/pkg/flag"
	"github.com/rancher/exporter-gateway/pkg/roundtripper"
)

type Endpoint struct {
	name   string
	target string
	client *http.Client
}

func (e *Endpoint) GetName() string {
	return e.name
}

func (e *Endpoint) Scrape(ctx context.Context, timeout time.Duration) (map[string]*dto.MetricFamily, error) {
	url := e.target

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/openmetrics-text; version=0.0.1,text/plain;version=0.0.4;q=0.5,*/*;q=0.1")
	req.Header.Add("Accept-Encoding", "gzip")
	req.Header.Set("User-Agent", "exporter-gateway")
	req.Header.Set("X-Prometheus-Scrape-Timeout-Seconds", fmt.Sprintf("%f", timeout.Seconds()))

	ctx, cancelFn := context.WithTimeout(ctx, timeout)
	defer cancelFn()

	resp, err := e.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			// don't care what error is it
			return nil, errors.Errorf("unexpected status code %d while sending to %s", resp.StatusCode, url)
		}

		return nil, errors.Errorf("unexpected status code %d while sending to %s: %s", resp.StatusCode, url, body)
	}

	responseBody := &bytes.Buffer{}
	if resp.Header.Get("Content-Encoding") != "gzip" {
		_, err = io.Copy(responseBody, resp.Body)
	} else {
		gzipReader, err := gzip.NewReader(bufio.NewReader(resp.Body))
		if err != nil {
			return nil, errors.Annotatef(err, "cannot uncompress zip response body from %s", url)
		}
		defer gzipReader.Close()

		_, err = io.Copy(responseBody, gzipReader)
	}
	if err != nil {
		return nil, errors.Annotatef(err, "cannot read response body from %s", url)
	}

	parser := &expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(responseBody)
	if err != nil {
		return nil, errors.Annotatef(err, "cannot parse response body from %s", url)
	}

	return metricFamilies, nil
}

func NewEndpoint(route flag.Router) (*Endpoint, error) {
	u, err := url.Parse(route.URL)
	if err != nil {
		return nil, errors.Annotate(err, "cannot parse URL")
	}
	rt, err := roundtripper.CreateRoundTripper(route)
	if err != nil {
		return nil, errors.Annotate(err, "cannot get http.RoundTripper")
	}

	return &Endpoint{
		route.Name,
		u.String(),
		&http.Client{Transport: rt},
	}, nil
}
