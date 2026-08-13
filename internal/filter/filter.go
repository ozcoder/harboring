package filter

import (
	"regexp"
	"strings"

	"github.com/harboring/harboring/internal/har"
)

type Filter struct {
	Method     string
	StatusMin  int
	StatusMax  int
	URLPattern string
	urlRegex   *regexp.Regexp
}

func New() *Filter {
	return &Filter{}
}

func (f *Filter) SetMethod(method string) {
	f.Method = strings.ToUpper(method)
}

func (f *Filter) SetStatusRange(min, max int) {
	f.StatusMin = min
	f.StatusMax = max
}

func (f *Filter) SetURLPattern(pattern string) error {
	if pattern == "" {
		f.urlRegex = nil
		f.URLPattern = ""
		return nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	f.urlRegex = re
	f.URLPattern = pattern
	return nil
}

func (f *Filter) Clear() {
	f.Method = ""
	f.StatusMin = 0
	f.StatusMax = 0
	f.URLPattern = ""
	f.urlRegex = nil
}

func (f *Filter) Matches(entry *har.EntryDetails) bool {
	if f.Method != "" {
		if strings.ToUpper(entry.Method) != f.Method {
			return false
		}
	}

	if f.StatusMin != 0 && entry.Status < f.StatusMin {
		return false
	}
	if f.StatusMax != 0 && entry.Status > f.StatusMax {
		return false
	}

	if f.urlRegex != nil {
		if !f.urlRegex.MatchString(entry.URL) {
			return false
		}
	}

	return true
}
