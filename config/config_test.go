package config

import (
	"io/ioutil"
	"net"
	"os"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func Test_NewConfig(t *testing.T) {
	err := os.Chdir("../testdata")
	assert.NoError(t, err)

	cfg := NewConfig()

	assert.Equal(t, uint16(55555), cfg.Port)
	assert.Len(t, cfg.Upstream.ExternalResolvers, 3)
	assert.Equal(t, "8.8.8.8", cfg.Upstream.ExternalResolvers[0].Host)
	assert.Equal(t, "8.8.4.4", cfg.Upstream.ExternalResolvers[1].Host)
	assert.Equal(t, "1.1.1.1", cfg.Upstream.ExternalResolvers[2].Host)
	assert.Len(t, cfg.CustomDNS.Mapping, 1)
	assert.Equal(t, net.ParseIP("192.168.178.3"), cfg.CustomDNS.Mapping["my.duckdns.org"])
	assert.Len(t, cfg.Conditional.Mapping, 1)
	assert.Equal(t, "192.168.178.1", cfg.ClientLookup.Upstream.Host)
	assert.Equal(t, []uint{2, 1}, cfg.ClientLookup.SingleNameOrder)
	assert.Len(t, cfg.Blocking.BlackLists, 2)
	assert.Len(t, cfg.Blocking.WhiteLists, 1)
	assert.Len(t, cfg.Blocking.ClientGroupsBlock, 2)
	assert.Equal(t, 0, cfg.Caching.MaxCachingTime)
	assert.Equal(t, 0, cfg.Caching.MinCachingTime)
}

func Test_NewConfig_Malformed(t *testing.T) {
	dir, err := ioutil.TempDir("", "blocky")
	defer os.Remove(dir)
	assert.NoError(t, err)
	err = os.Chdir(dir)
	assert.NoError(t, err)
	err = ioutil.WriteFile("config.yml", []byte("malformed_config"), 0644)
	assert.NoError(t, err)

	defer func() { logrus.StandardLogger().ExitFunc = nil }()

	var fatal bool

	logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

	_ = NewConfig()

	assert.True(t, fatal)
}

func Test_NewConfig_FileDoesNotExist(t *testing.T) {
	err := os.Chdir("../..")
	assert.NoError(t, err)

	defer func() { logrus.StandardLogger().ExitFunc = nil }()

	var fatal bool

	logrus.StandardLogger().ExitFunc = func(int) { fatal = true }
	_ = NewConfig()

	assert.True(t, fatal)
}

var tests = []struct {
	name       string
	args       string
	wantResult Upstream
	wantErr    bool
}{
	{
		name:       "udpWithPort",
		args:       "udp:4.4.4.4:531",
		wantResult: Upstream{Net: "udp", Host: "4.4.4.4", Port: 531},
	},
	{
		name:       "udpDefault",
		args:       "udp:4.4.4.4",
		wantResult: Upstream{Net: "udp", Host: "4.4.4.4", Port: 53},
	},
	{
		name:       "tcpWithPort",
		args:       "tcp:4.4.4.4:4711",
		wantResult: Upstream{Net: "tcp", Host: "4.4.4.4", Port: 4711},
	},
	{
		name:       "tcpDefault",
		args:       "tcp:4.4.4.4",
		wantResult: Upstream{Net: "tcp", Host: "4.4.4.4", Port: 53},
	},
	{
		name:       "tcpTlsDefault",
		args:       "tcp-tls:4.4.4.4",
		wantResult: Upstream{Net: "tcp-tls", Host: "4.4.4.4", Port: 853},
	},
	{
		name:       "dohDefault",
		args:       "https:4.4.4.4",
		wantResult: Upstream{Net: "https", Host: "4.4.4.4", Port: 443},
	},
	{
		name:       "dohWithPort",
		args:       "https:4.4.4.4:888",
		wantResult: Upstream{Net: "https", Host: "4.4.4.4", Port: 888},
	},
	{
		name:       "dohNamed",
		args:       "https://dns.google/dns-query",
		wantResult: Upstream{Net: "https", Host: "dns.google", Port: 443, Path: "/dns-query"},
	},
	{
		name:       "dohNamedMultiSlash",
		args:       "https://dns.google/dns-query/a/b",
		wantResult: Upstream{Net: "https", Host: "dns.google", Port: 443, Path: "/dns-query/a/b"},
	},
	{
		name:       "dohNamedWithPort",
		args:       "https://dns.google:888/dns-query",
		wantResult: Upstream{Net: "https", Host: "dns.google", Port: 888, Path: "/dns-query"},
	},
	{
		name:       "empty",
		args:       "",
		wantResult: Upstream{},
	},
	{
		name:    "withoutHost",
		args:    "tcp::53",
		wantErr: true,
	},
	{
		name:    "withoutNet",
		args:    ":1.1.1.1:53",
		wantErr: true,
	},
	{
		name:    "negativePort",
		args:    "tcp:4.4.4.4:-1",
		wantErr: true,
	},
	{
		name:    "invalidPort",
		args:    "tcp:4.4.4.4:65536",
		wantErr: true,
	},
	{
		name:    "notNumericPort",
		args:    "tcp:4.4.4.4:A53",
		wantErr: true,
	},
	{
		name:    "wrongProtocol",
		args:    "bla:4.4.4.4:53",
		wantErr: true,
	},
	{
		name:    "wrongFormat",
		args:    "tcp-4.4.4.4",
		wantErr: true,
	},
}

func Test_parseUpstream(t *testing.T) {
	for _, tt := range tests {
		rr := tt
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := ParseUpstream(rr.args)
			if (err != nil) != rr.wantErr {
				t.Errorf("parseUpstream() error = %v, wantErr %v", err, rr.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, rr.wantResult) {
				t.Errorf("parseUpstream() = %v, want %v", gotResult, rr.wantResult)
			}
		})
	}
}
