package pushgateway

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

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

func (e *Endpoint) do(ctx context.Context, method string, job string, grouping map[string]string, mfs map[string]*dto.MetricFamily) error {
	if len(job) == 0 {
		return errors.New("cannot push to an unknown job")
	}
	if grouping == nil {
		grouping = make(map[string]string, 0)
	}

	urlComponents := []string{url.QueryEscape(job)}
	for ln, lv := range grouping {
		urlComponents = append(urlComponents, ln, lv)
	}
	url := fmt.Sprintf("%s/job/%s", e.target, strings.Join(urlComponents, "/"))

	requestBody := &bytes.Buffer{}
	requestBodyEnc := expfmt.NewEncoder(requestBody, expfmt.FmtProtoDelim)
	// clean metrics labels
	for _, mf := range mfs {
		if mf == nil {
			continue
		}

		for _, m := range mf.GetMetric() {
			if m == nil {
				continue
			}

			oldLabel := m.GetLabel()
			newLabel := make([]*dto.LabelPair, 0, len(oldLabel))

			for _, l := range oldLabel {
				isJobLabel := l.GetName() == "job"
				_, isGroupingLabel := grouping[l.GetName()]

				if !isJobLabel && !isGroupingLabel {
					newLabel = append(newLabel, l)
				}
			}

			m.Label = newLabel
		}

		requestBodyEnc.Encode(mf)
	}

	req, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", string(expfmt.FmtProtoDelim))

	resp, err := e.client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			// don't care what error is it
			return errors.Errorf("unexpected status code %d while sending to %s", resp.StatusCode, url)
		}

		return errors.Errorf("unexpected status code %d while sending to %s: %s", resp.StatusCode, url, body)
	}

	return nil
}

func (e *Endpoint) Put(ctx context.Context, job string, grouping map[string]string, mfs map[string]*dto.MetricFamily) error {
	return e.do(ctx, http.MethodPut, job, grouping, mfs)
}

func (e *Endpoint) Delete(ctx context.Context, job string, grouping map[string]string) error {
	return e.do(ctx, http.MethodDelete, job, grouping, nil)
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
