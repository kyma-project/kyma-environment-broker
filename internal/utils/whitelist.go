package utils

import (
	"sort"
	"strings"
)

type Whitelist map[string]struct{}

func (t *Whitelist) Unmarshal(s string) error {
	*t = make(Whitelist)

	for _, item := range strings.Split(s, ";") {
		(*t)[item] = struct{}{}
	}

	return nil
}

func (t *Whitelist) Contains(item string) bool {
	_, found := (*t)[item]
	return found
}

func (t Whitelist) String() string {
	keys := make([]string, 0, len(t))
	for item := range t {
		keys = append(keys, item)
	}

	sort.Strings(keys)

	output := ""
	for _, item := range keys {
		output += item + ";"
	}

	return output
}
