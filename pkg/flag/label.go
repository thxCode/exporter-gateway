package flag

import (
	"strings"
)

type Labels []string

func (f *Labels) Set(value string) error {
	*f = append(*f, value)

	return nil
}

func (f *Labels) String() string {
	return strings.Join(*f, ",")
}

func (f *Labels) Unwrap() map[string]string {
	ret := make(map[string]string, len(*f))

	for _, l := range *f {
		ls := strings.SplitN(l, "=", 2)
		if len(ls) == 2 && ls[0] != "job" {
			ret[strings.TrimSpace(ls[0])] = strings.TrimSpace(ls[1])
		}
	}

	return ret
}
