// Package dnsmasq_exporter embeds https://github.com/google/dnsmasq_exporter
package dnsmasq_exporter //nolint:golint

import (
	"github.com/go-kit/log"
	"github.com/google/dnsmasq_exporter/collector"
	"github.com/grafana/agent/pkg/integrations"
	integrations_v2 "github.com/grafana/agent/pkg/integrations/v2"
	"github.com/grafana/agent/pkg/integrations/v2/metricsutils"
	"github.com/miekg/dns"
)

// DefaultConfig is the default config for dnsmasq_exporter.
var DefaultConfig Config = Config{
	DnsmasqAddress: "localhost:53",
	LeasesPath:     "/var/lib/misc/dnsmasq.leases",
}

// Config controls the dnsmasq_exporter integration.
type Config struct {
	// DnsmasqAddress is the address of the dnsmasq server (host:port).
	DnsmasqAddress string `yaml:"dnsmasq_address,omitempty"`

	// Path to the dnsmasq leases file.
	LeasesPath string `yaml:"leases_path,omitempty"`
}

// Name returns the name of the integration that this config is for.
func (c *Config) Name() string {
	return "dnsmasq_exporter"
}

// InstanceKey returns the address of the dnsmasq server.
func (c *Config) InstanceKey(agentKey string) (string, error) {
	return c.DnsmasqAddress, nil
}

// NewIntegration converts this config into an instance of an integration.
func (c *Config) NewIntegration(l log.Logger) (integrations.Integration, error) {
	return New(l, c)
}

// UnmarshalYAML implements yaml.Unmarshaler for Config.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultConfig

	type plain Config
	return unmarshal((*plain)(c))
}

func init() {
	integrations.RegisterIntegration(&Config{})
	integrations_v2.RegisterLegacy(&Config{}, integrations_v2.TypeMultiplex, metricsutils.CreateShim)
}

// New creates a new dnsmasq_exporter integration. The integration scrapes metrics
// from a dnsmasq server.
func New(log log.Logger, c *Config) (integrations.Integration, error) {
	exporter := collector.New(log, &dns.Client{
		SingleInflight: true,
	}, c.DnsmasqAddress, c.LeasesPath)

	return integrations.NewCollectorIntegration(c.Name(), integrations.WithCollectors(exporter)), nil
}
