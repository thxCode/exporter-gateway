package roundtripper

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/juju/errors"
	"github.com/rancher/exporter-gateway/pkg/flag"
)

func CreateRoundTripper(f flag.Router) (http.RoundTripper, error) {
	// parse tlsConfig
	var tlsConfig *tls.Config
	if f.TLSConfig != nil {
		config := f.TLSConfig

		if len(config.CertFile) != 0 && len(config.KeyFile) != 0 {
			tlsConfig = &tls.Config{}

			tlsCert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
			if err != nil {
				return nil, errors.Annotate(err, "unable to parse TLS key pair")
			}

			tlsConfig.Certificates = []tls.Certificate{tlsCert}
		}

		if len(config.CAFile) != 0 {
			if tlsConfig == nil {
				tlsConfig = &tls.Config{}
			}

			caPEMBlock, err := ioutil.ReadFile(config.CAFile)
			if err != nil {
				return nil, errors.Annotate(err, "unable to read client CA file")
			}
			if len(caPEMBlock) == 0 {
				return nil, errors.New("unable to load client CA file: empty file")
			}
			caPool := x509.NewCertPool()
			if ok := caPool.AppendCertsFromPEM(caPEMBlock); !ok {
				return nil, errors.New("unable to load client CA file")
			}

			tlsConfig.RootCAs = caPool
			tlsConfig.InsecureSkipVerify = config.InsecureSkipVerify
		}
	}

	// wrap roundTripper
	var rt http.RoundTripper = &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	if len(f.BearerToken) > 0 {
		rt = &bearerAuthRoundTripper{f.BearerToken, rt}
	} else if len(f.BearerTokenFile) > 0 {
		rt = &bearerAuthFileRoundTripper{f.BearerTokenFile, rt}
	}
	if f.BasicAuth != nil {
		ba := f.BasicAuth
		rt = &basicAuthRoundTripper{ba.Username, ba.Password, ba.PasswordFile, rt}
	}

	return rt, nil
}

type bearerAuthRoundTripper struct {
	bearerToken flag.Secret
	rt          http.RoundTripper
}

func (rt *bearerAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(req.Header.Get("Authorization")) == 0 {
		req = cloneRequest(req)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", string(rt.bearerToken)))
	}
	return rt.rt.RoundTrip(req)
}

type bearerAuthFileRoundTripper struct {
	bearerFile string
	rt         http.RoundTripper
}

func (rt *bearerAuthFileRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(req.Header.Get("Authorization")) == 0 {
		b, err := ioutil.ReadFile(rt.bearerFile)
		if err != nil {
			return nil, errors.Errorf("unable to read bearer token file %s: %s", rt.bearerFile, err)
		}
		bearerToken := strings.TrimSpace(string(b))

		req = cloneRequest(req)
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	return rt.rt.RoundTrip(req)
}

type basicAuthRoundTripper struct {
	username     string
	password     flag.Secret
	passwordFile string
	rt           http.RoundTripper
}

func (rt *basicAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(req.Header.Get("Authorization")) != 0 {
		return rt.rt.RoundTrip(req)
	}
	req = cloneRequest(req)
	if rt.passwordFile != "" {
		bs, err := ioutil.ReadFile(rt.passwordFile)
		if err != nil {
			return nil, errors.Errorf("unable to read basic auth password file %s: %s", rt.passwordFile, err)
		}
		req.SetBasicAuth(rt.username, strings.TrimSpace(string(bs)))
	} else {
		req.SetBasicAuth(rt.username, strings.TrimSpace(string(rt.password)))
	}
	return rt.rt.RoundTrip(req)
}

func cloneRequest(r *http.Request) *http.Request {
	// Shallow copy of the struct.
	r2 := new(http.Request)
	*r2 = *r
	// Deep copy of the Header.
	r2.Header = make(http.Header)
	for k, s := range r.Header {
		r2.Header[k] = s
	}
	return r2
}
