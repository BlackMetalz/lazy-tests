package scenario

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	VersionV1                 = "v1"
	ProtocolTCP               = "tcp"
	ProtocolMySQL             = "mysql"
	ProtocolRedis             = "redis"
	ProtocolPostgres          = "postgres"
	PatternHoldOpen           = "hold-open"
	PatternConnectChurn       = "connect-churn"
	defaultConnectTimeout     = 2 * time.Second
	defaultConnectionsCap     = 5000
	defaultReportDirectory    = "./reports"
	defaultPrivateNetworkOnly = true
	defaultPrometheusAddr     = "127.0.0.1:2112"
)

var supportedProtocols = map[string]struct{}{
	ProtocolTCP:      {},
	ProtocolMySQL:    {},
	ProtocolRedis:    {},
	ProtocolPostgres: {},
}

type Duration time.Duration

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode {
		return fmt.Errorf("duration must be scalar")
	}

	parsed, err := time.ParseDuration(strings.TrimSpace(value.Value))
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", value.Value, err)
	}

	*d = Duration(parsed)
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d Duration) String() string {
	return time.Duration(d).String()
}

func (d Duration) Value() time.Duration {
	return time.Duration(d)
}

type Scenario struct {
	Version    string     `yaml:"version" json:"version"`
	Name       string     `yaml:"name" json:"name"`
	Protocol   string     `yaml:"protocol" json:"protocol"`
	Target     Target     `yaml:"target" json:"target"`
	Auth       Auth       `yaml:"auth" json:"auth"`
	Workload   Workload   `yaml:"workload" json:"workload"`
	Timeouts   Timeouts   `yaml:"timeouts" json:"timeouts"`
	Assertions Assertions `yaml:"assertions" json:"assertions"`
	Safety     Safety     `yaml:"safety" json:"safety"`
	Output     Output     `yaml:"output" json:"output"`
}

type Target struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
}

type Auth struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password,omitempty"`
	Database string `yaml:"database" json:"database,omitempty"`
	RedisDB  int    `yaml:"redis_db" json:"redis_db,omitempty"`
}

type Workload struct {
	Pattern           string   `yaml:"pattern" json:"pattern"`
	Connections       int      `yaml:"connections" json:"connections"`
	ConnectRatePerSec int      `yaml:"connect_rate_per_sec" json:"connect_rate_per_sec"`
	Duration          Duration `yaml:"duration" json:"duration"`
	HoldTime          Duration `yaml:"hold_time" json:"hold_time"`
}

type Timeouts struct {
	Connect Duration `yaml:"connect" json:"connect"`
}

type Assertions struct {
	MaxErrorRatePct float64 `yaml:"max_error_rate_pct" json:"max_error_rate_pct"`
	MaxP95ConnectMs float64 `yaml:"max_p95_connect_ms" json:"max_p95_connect_ms"`
}

type Safety struct {
	MaxConnectionsCap int   `yaml:"max_connections_cap" json:"max_connections_cap"`
	PrivateOnly       *bool `yaml:"private_network_only" json:"private_network_only"`
}

type Output struct {
	ReportDir  string           `yaml:"report_dir" json:"report_dir"`
	Prometheus PrometheusOutput `yaml:"prometheus" json:"prometheus"`
}

type PrometheusOutput struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	ListenAddr string `yaml:"listen_addr" json:"listen_addr"`
}

type Overrides struct {
	Host              string
	Port              int
	Duration          string
	Connections       int
	ConnectRatePerSec int
	OutDir            string
}

func Parse(data []byte) (*Scenario, error) {
	var s Scenario
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse scenario yaml: %w", err)
	}

	s.applyDefaults()
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

func Load(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read scenario file %s: %w", path, err)
	}

	return Parse(data)
}

func (s *Scenario) ApplyOverrides(o Overrides) error {
	if strings.TrimSpace(o.Host) != "" {
		s.Target.Host = strings.TrimSpace(o.Host)
	}

	if o.Port > 0 {
		s.Target.Port = o.Port
	}

	if o.Duration != "" {
		parsed, err := time.ParseDuration(o.Duration)
		if err != nil {
			return fmt.Errorf("invalid duration override %q: %w", o.Duration, err)
		}
		s.Workload.Duration = Duration(parsed)
	}

	if o.Connections > 0 {
		s.Workload.Connections = o.Connections
	}

	if o.ConnectRatePerSec > 0 {
		s.Workload.ConnectRatePerSec = o.ConnectRatePerSec
	}

	if o.OutDir != "" {
		s.Output.ReportDir = o.OutDir
	}

	s.applyDefaults()
	return s.Validate()
}

func (s *Scenario) Validate() error {
	if strings.TrimSpace(s.Version) == "" {
		return errors.New("version is required")
	}
	if s.Version != VersionV1 {
		return fmt.Errorf("unsupported version %q", s.Version)
	}

	if strings.TrimSpace(s.Name) == "" {
		return errors.New("name is required")
	}

	if _, ok := supportedProtocols[s.Protocol]; !ok {
		return fmt.Errorf("unsupported protocol %q", s.Protocol)
	}

	if strings.TrimSpace(s.Target.Host) == "" {
		return errors.New("target.host is required")
	}
	if s.Target.Port <= 0 || s.Target.Port > 65535 {
		return errors.New("target.port must be between 1 and 65535")
	}

	switch s.Workload.Pattern {
	case PatternHoldOpen, PatternConnectChurn:
	default:
		return fmt.Errorf("unsupported workload.pattern %q", s.Workload.Pattern)
	}

	if s.Workload.Connections <= 0 {
		return errors.New("workload.connections must be > 0")
	}
	if s.Workload.ConnectRatePerSec <= 0 {
		return errors.New("workload.connect_rate_per_sec must be > 0")
	}
	if s.Workload.Duration.Value() <= 0 {
		return errors.New("workload.duration must be > 0")
	}
	if s.Workload.HoldTime.Value() < 0 {
		return errors.New("workload.hold_time must be >= 0")
	}

	if s.Timeouts.Connect.Value() <= 0 {
		return errors.New("timeouts.connect must be > 0")
	}

	if s.Assertions.MaxErrorRatePct < 0 {
		return errors.New("assertions.max_error_rate_pct must be >= 0")
	}
	if s.Assertions.MaxP95ConnectMs < 0 {
		return errors.New("assertions.max_p95_connect_ms must be >= 0")
	}

	if s.Safety.MaxConnectionsCap <= 0 {
		return errors.New("safety.max_connections_cap must be > 0")
	}
	if s.Workload.Connections > s.Safety.MaxConnectionsCap {
		return fmt.Errorf("workload.connections %d exceeds safety.max_connections_cap %d", s.Workload.Connections, s.Safety.MaxConnectionsCap)
	}

	if s.Protocol == ProtocolRedis && s.Auth.RedisDB < 0 {
		return errors.New("auth.redis_db must be >= 0")
	}

	if strings.TrimSpace(s.Output.ReportDir) == "" {
		return errors.New("output.report_dir cannot be empty")
	}
	if s.Output.Prometheus.Enabled && strings.TrimSpace(s.Output.Prometheus.ListenAddr) == "" {
		return errors.New("output.prometheus.listen_addr cannot be empty when prometheus is enabled")
	}

	return nil
}

func (s *Scenario) applyDefaults() {
	if s.Timeouts.Connect.Value() <= 0 {
		s.Timeouts.Connect = Duration(defaultConnectTimeout)
	}

	if s.Safety.MaxConnectionsCap <= 0 {
		s.Safety.MaxConnectionsCap = defaultConnectionsCap
	}

	if s.Safety.PrivateOnly == nil {
		v := defaultPrivateNetworkOnly
		s.Safety.PrivateOnly = &v
	}

	if strings.TrimSpace(s.Output.ReportDir) == "" {
		s.Output.ReportDir = defaultReportDirectory
	}

	if strings.TrimSpace(s.Output.Prometheus.ListenAddr) == "" {
		s.Output.Prometheus.ListenAddr = defaultPrometheusAddr
	}
}

func (s *Scenario) PrivateNetworkOnly() bool {
	if s.Safety.PrivateOnly == nil {
		return defaultPrivateNetworkOnly
	}
	return *s.Safety.PrivateOnly
}

func SupportedProtocols() []string {
	return []string{ProtocolTCP, ProtocolMySQL, ProtocolRedis, ProtocolPostgres}
}

func IsPrivateHost(host string) (bool, error) {
	normalized := strings.TrimSpace(strings.ToLower(host))
	if normalized == "localhost" {
		return true, nil
	}

	if ip := net.ParseIP(host); ip != nil {
		return isPrivateIP(ip), nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return false, fmt.Errorf("resolve host %q: %w", host, err)
	}
	if len(ips) == 0 {
		return false, fmt.Errorf("resolve host %q: no IP addresses", host)
	}

	for _, ip := range ips {
		if !isPrivateIP(ip) {
			return false, nil
		}
	}

	return true, nil
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}

	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}

	// net.ParseIP always returns a 16-byte slice even for IPv4 addresses,
	// which makes AddrFromSlice return an IPv4-in-IPv6 form where IsPrivate()
	// won't match RFC 1918 ranges. Unmap() converts it to a plain IPv4 addr first.
	addr = addr.Unmap()

	if addr.IsPrivate() {
		return true
	}

	if addr.IsLinkLocalUnicast() {
		return true
	}

	if addr.Is6() {
		if addr.IsLoopback() {
			return true
		}
		ulaPrefix, _ := netip.ParsePrefix("fc00::/7")
		if ulaPrefix.Contains(addr) {
			return true
		}
	}

	return false
}
