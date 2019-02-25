package flag

import (
	"encoding/json"
	"gopkg.in/yaml.v2"
	"k8s.io/helm/pkg/strvals"
	"strings"
)

type Secret string

func (f Secret) MarshalJSON() ([]byte, error) {
	if f != "" {
		return []byte("<secret>"), nil
	}

	return []byte{}, nil
}

func (f *Secret) UnmarshalJSON(body []byte) error {
	if len(body) != 0 {
		*f = Secret(string(body))
	}
	return nil
}

type BasicAuth struct {
	Username     string `json:"username" yaml:"username"`
	Password     Secret `json:"password,omitempty" yaml:"password,omitempty"`
	PasswordFile string `json:"passwordFile,omitempty" yaml:"passwordFile,omitempty"`
}

type TLSConfig struct {
	CAFile             string `json:"caFile,omitempty" yaml:"caFile,omitempty"`
	CertFile           string `json:"certFile,omitempty" yaml:"certFile,omitempty"`
	KeyFile            string `json:"keyFile,omitempty" yaml:"keyFile,omitempty"`
	ServerName         string `json:"serverName,omitempty" yaml:"serverName,omitempty"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify" yaml:"insecureSkipVerify"`
}

type Router struct {
	Name            string     `json:"name" yaml:"name"`
	URL             string     `json:"url" yaml:"url"`
	BasicAuth       *BasicAuth `json:"basicAuth,omitempty" yaml:"basicAuth,omitempty"`
	BearerToken     Secret     `json:"bearerToken,omitempty" yaml:"bearerToken,omitempty"`
	BearerTokenFile string     `json:"bearerTokenFile,omitempty" yaml:"bearerTokenFile,omitempty"`
	TLSConfig       *TLSConfig `json:"tlsConfig,omitempty" yaml:"tlsConfig,omitempty"`
}

type Routers []string

func (f *Routers) Set(value string) error {
	*f = append(*f, value)

	return nil
}

func (f *Routers) String() string {
	bs, err := json.Marshal(f.Unwrap())
	if err != nil {
		return ""
	}

	return string(bs)
}

func (f *Routers) Unwrap() []Router {
	ret := make([]Router, 0)

	tmpRouteMap := make(map[string]interface{})
	for _, value := range *f {
		// [x.y.z,x.a.b] -> {x:{y:z,a:b}}, so we don't care the parsed error
		strvals.ParseIntoString(value, tmpRouteMap)
	}
	if bs, err := yaml.Marshal(tmpRouteMap); err == nil {
		routeMap := make(map[string]Router, len(tmpRouteMap))
		// map[string]interface{} -> map[string]Router, so we don't care the unmarshalled error
		yaml.Unmarshal(bs, routeMap)

		for n, route := range routeMap {
			if len(route.URL) != 0 {
				// trim right /
				route.URL = strings.TrimRight(route.URL, "/")
				route.Name = n
				ret = append(ret, route)
			}
		}
	}

	return ret
}
