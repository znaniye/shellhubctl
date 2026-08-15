package selector

import (
	"reflect"
	"strings"
	"testing"

	"github.com/znaniye/shellhubctl/internal/shellhub"
)

func testDevice(uid, name, prettyName string, online bool, tags ...string) shellhub.Device {
	device := shellhub.Device{
		UID:    uid,
		Name:   name,
		Online: online,
		Info:   shellhub.DeviceInfo{PrettyName: prettyName},
	}

	for _, tag := range tags {
		device.Tags = append(device.Tags, shellhub.Tag{Name: tag})
	}

	return device
}

func mustCompile(t *testing.T, opts Options) *Matcher {
	t.Helper()

	m, err := opts.Compile()
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	return m
}

func uids(devices []shellhub.Device) []string {
	out := make([]string, 0, len(devices))

	for _, device := range devices {
		out = append(out, device.UID)
	}

	return out
}

func TestApplyNameFilter(t *testing.T) {
	devices := []shellhub.Device{
		testDevice("d1", "my-web-01", "Ubuntu 22.04", true),
		testDevice("d2", "db-server", "Debian 12", true),
		testDevice("d3", "WEB-frontend", "Alpine", false),
	}

	m := mustCompile(t, Options{Names: []string{"web"}})

	if got := uids(m.Apply(devices)); !reflect.DeepEqual(got, []string{"d1", "d3"}) {
		t.Errorf("Apply() = %v, want [d1 d3]", got)
	}
}

func TestApplyDistroFilterUsesPrettyNameOnly(t *testing.T) {
	devices := []shellhub.Device{
		testDevice("d1", "srv-1", "Ubuntu 22.04", true),
		testDevice("d2", "srv-2", "Debian 12", true),
		testDevice("d3", "srv-3", "", true),
	}

	devices[2].Info.ID = "ubuntu-22"

	m := mustCompile(t, Options{Distros: []string{"ubuntu"}})

	if got := uids(m.Apply(devices)); !reflect.DeepEqual(got, []string{"d1"}) {
		t.Errorf("Apply() = %v, want [d1]", got)
	}
}

func TestApplyTagFilterRequiresAllAndExact(t *testing.T) {
	devices := []shellhub.Device{
		testDevice("d1", "srv-1", "Ubuntu", true, "web", "prod"),
		testDevice("d2", "srv-2", "Debian", true, "web"),
		testDevice("d3", "srv-3", "Alpine", true, "webserver", "prod"),
		testDevice("d4", "srv-4", "Fedora", true, "prod", "web", "eu"),
	}

	m := mustCompile(t, Options{Tags: []string{"web", "prod"}})

	if got := uids(m.Apply(devices)); !reflect.DeepEqual(got, []string{"d1", "d4"}) {
		t.Errorf("Apply() = %v, want [d1 d4]", got)
	}
}

func TestApplyOnlineFilter(t *testing.T) {
	devices := []shellhub.Device{
		testDevice("d1", "srv-1", "Ubuntu", true),
		testDevice("d2", "srv-2", "Debian", false),
		testDevice("d3", "srv-3", "Alpine", true),
	}

	m := mustCompile(t, Options{Online: true})

	if got := uids(m.Apply(devices)); !reflect.DeepEqual(got, []string{"d1", "d3"}) {
		t.Errorf("Apply() = %v, want [d1 d3]", got)
	}
}

func TestApplyAndCombinesCriteria(t *testing.T) {
	devices := []shellhub.Device{
		testDevice("d1", "web-01", "Ubuntu 22.04", false),
		testDevice("d2", "db-01", "Ubuntu 22.04", true),
		testDevice("d3", "web-02", "Debian 12", true),
		testDevice("d4", "web-03", "Ubuntu 22.04", true),
	}

	m := mustCompile(t, Options{
		Names:   []string{"web"},
		Distros: []string{"ubuntu"},
		Online:  true,
	})

	if got := uids(m.Apply(devices)); !reflect.DeepEqual(got, []string{"d4"}) {
		t.Errorf("Apply() = %v, want [d4]", got)
	}
}

func TestApplyEmptyOptionsPassesAllInOrder(t *testing.T) {
	devices := []shellhub.Device{
		testDevice("d1", "web-01", "Ubuntu", true),
		testDevice("d2", "db-01", "Debian", false),
	}

	m := mustCompile(t, Options{})

	if got := m.Apply(devices); !reflect.DeepEqual(got, devices) {
		t.Errorf("Apply() = %v, want %v", got, devices)
	}
}

func TestApplyNoMatchReturnsEmptyNonNil(t *testing.T) {
	devices := []shellhub.Device{
		testDevice("d1", "web-01", "Ubuntu", true),
	}

	m := mustCompile(t, Options{Names: []string{"nomatch"}})

	got := m.Apply(devices)
	if got == nil {
		t.Fatal("Apply() = nil, want non-nil empty slice")
	}

	if len(got) != 0 {
		t.Errorf("Apply() len = %d, want 0", len(got))
	}
}

func TestApplyInlineFlagInPattern(t *testing.T) {
	devices := []shellhub.Device{
		testDevice("d1", "foo\nbar", "Ubuntu", true),
		testDevice("d2", "foo.bar", "Ubuntu", true),
	}

	m := mustCompile(t, Options{Names: []string{"(?s)foo.bar"}})

	if got := uids(m.Apply(devices)); !reflect.DeepEqual(got, []string{"d1", "d2"}) {
		t.Errorf("Apply() = %v, want [d1 d2]", got)
	}
}

func TestCompileEmptyPatternMatchesEverything(t *testing.T) {
	devices := []shellhub.Device{
		testDevice("d1", "web-01", "Ubuntu", true),
		testDevice("d2", "db-01", "Debian", false),
	}

	m := mustCompile(t, Options{Names: []string{""}})

	if got := uids(m.Apply(devices)); !reflect.DeepEqual(got, []string{"d1", "d2"}) {
		t.Errorf("Apply() = %v, want [d1 d2]", got)
	}
}

func TestCompileInvalidNameRegex(t *testing.T) {
	_, err := (Options{Names: []string{"("}}).Compile()
	if err == nil {
		t.Fatal("Compile() error = nil, want invalid regex error")
	}

	if !strings.Contains(err.Error(), "(") {
		t.Errorf("Compile() error = %v, want it to name the pattern", err)
	}
}

func TestCompileInvalidDistroRegex(t *testing.T) {
	_, err := (Options{Distros: []string{"["}}).Compile()
	if err == nil {
		t.Fatal("Compile() error = nil, want invalid regex error")
	}

	if !strings.Contains(err.Error(), "[") {
		t.Errorf("Compile() error = %v, want it to name the pattern", err)
	}
}

func TestCompileInvalidRegexDoesNotPanic(t *testing.T) {
	for _, opts := range []Options{
		{Names: []string{"*"}},
		{Distros: []string{"(?x"}},
	} {
		_, err := opts.Compile()
		if err == nil {
			t.Errorf("Compile() with %+v error = nil, want error", opts)
		}
	}
}
