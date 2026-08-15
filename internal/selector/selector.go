package selector

import (
	"fmt"
	"regexp"

	"github.com/znaniye/shellhubctl/internal/shellhub"
)

type Options struct {
	Names   []string
	Distros []string
	Tags    []string
	Online  bool
}

type Matcher struct {
	names   []*regexp.Regexp
	distros []*regexp.Regexp
	tags    []string
	online  bool
}

func (o Options) Compile() (*Matcher, error) {
	names, err := compilePatterns(o.Names)
	if err != nil {
		return nil, err
	}

	distros, err := compilePatterns(o.Distros)
	if err != nil {
		return nil, err
	}

	return &Matcher{
		names:   names,
		distros: distros,
		tags:    o.Tags,
		online:  o.Online,
	}, nil
}

func compilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))

	for _, pattern := range patterns {
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %q: %w", pattern, err)
		}

		compiled = append(compiled, re)
	}

	return compiled, nil
}

func (m *Matcher) Apply(devices []shellhub.Device) []shellhub.Device {
	filtered := make([]shellhub.Device, 0, len(devices))

	for _, device := range devices {
		if m.match(device) {
			filtered = append(filtered, device)
		}
	}

	return filtered
}

func (m *Matcher) match(device shellhub.Device) bool {
	for _, re := range m.names {
		if !re.MatchString(device.Name) {
			return false
		}
	}

	for _, re := range m.distros {
		if !re.MatchString(device.Info.PrettyName) {
			return false
		}
	}

	if len(m.tags) > 0 {
		names := device.TagNames()

		for _, tag := range m.tags {
			if !hasTag(names, tag) {
				return false
			}
		}
	}

	if m.online && !device.Online {
		return false
	}

	return true
}

func hasTag(names []string, tag string) bool {
	for _, name := range names {
		if name == tag {
			return true
		}
	}

	return false
}
