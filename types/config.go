package types

import (
	"github.com/go-rod/rod-mcp/utils"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

const ConfigName = "rod-mcp.yaml"

type Config struct {
	Mode             Mode              `yaml:"mode" json:"mode"`
	CDPEndpoint      string            `yaml:"cdpEndpoint" json:"cdpEndpoint"`
	ServerName       string            `yaml:"serverName" json:"serverName"`
	ServerVersion    string            `yaml:"-" json:"-"`
	BrowserBinPath   string            `yaml:"browserBinPath" json:"browserBinPath"`
	Headless         bool              `yaml:"headless" json:"headless"`
	BrowserTempDir   string            `yaml:"browserTempDir" json:"browserTempDir"`
	NoSandbox        bool              `yaml:"noSandbox" json:"noSandbox"`
	Proxy            string            `yaml:"proxy" json:"proxy"`
	LoggerConfig     LoggerConfig      `yaml:"loggerConfig" json:"loggerConfig"`
	ExtraHTTPHeaders map[string]string `yaml:"extraHTTPHeaders" json:"extraHTTPHeaders"`
	CompactSnapshot  bool              `yaml:"compactSnapshot" json:"compactSnapshot"`
	// DomainHeaders maps domain patterns to headers that should be injected for matching URLs.
	// Patterns support wildcards: "*.example.com" matches "www.example.com", "api.example.com", etc.
	// Headers from matching patterns are merged with ExtraHTTPHeaders.
	DomainHeaders map[string]map[string]string `yaml:"domainHeaders" json:"domainHeaders"`
}

var (
	DefaultBrowserTempDir = "./rod/browser"
	DefaultServerName     = "Rod Server"

	DefaultConfig = Config{
		BrowserBinPath: "",
		Headless:       false,
		BrowserTempDir: DefaultBrowserTempDir,
		NoSandbox:      false,
		Proxy:          "",
		ServerName:     DefaultServerName,
		LoggerConfig:   DefaultLoggerConfig,
		Mode:           Text,
	}
)

// InitDefaultConfig Generate the default configuration file
func InitDefaultConfig() error {

	// First, check if the configuration file exists at the default path. If it exists, do not generate the default configuration file.
	defaultConfigPath := filepath.Join("./", ConfigName)
	if exist, _ := utils.PathExists(defaultConfigPath); exist {
		return nil
	}

	// if default config file not exist, create it
	defaultConfig, err := os.Create(defaultConfigPath)
	if err != nil {
		return err
	}

	encoder := yaml.NewEncoder(defaultConfig)
	defer encoder.Close()

	err = encoder.Encode(DefaultConfig)
	if err != nil {
		return err
	}
	return nil
}

// LoadConfig Actually load the configuration file
// if ConfigPath is empty, generate the default configuration file in the current directory
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = filepath.Join("./", ConfigName)
		if err := InitDefaultConfig(); err != nil {
			return nil, errors.Wrapf(err, "init default config failed")
		}
	}

	// check if config file exist
	exist, err := utils.PathExists(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not open config file")
	}

	if exist {
		// validate config file name
		fileName := utils.FileName(configPath)
		if strings.Contains(ConfigName, fileName) {
			file, err := os.Open(configPath)
			if err != nil {
				return nil, err
			}
			defer file.Close()

			decoder := yaml.NewDecoder(file)
			var config Config
			if err := decoder.Decode(&config); err != nil {
				return nil, err
			}
			return &config, nil
		}
		return nil, errors.Wrapf(err, "config file name is wrong")
	}
	return nil, errors.Wrapf(err, "config path %s not found", configPath)
}

// GetHeadersForURL returns all headers that should be applied for the given URL.
// This merges ExtraHTTPHeaders with any matching DomainHeaders patterns.
func (c *Config) GetHeadersForURL(urlStr string) map[string]string {
	result := make(map[string]string)

	// Start with global headers
	for k, v := range c.ExtraHTTPHeaders {
		result[k] = v
	}

	// If no domain headers configured or URL is empty, return global headers only
	if len(c.DomainHeaders) == 0 || urlStr == "" {
		return result
	}

	// Extract host from URL
	host := extractHost(urlStr)
	if host == "" {
		return result
	}

	// Check each domain pattern
	for pattern, headers := range c.DomainHeaders {
		if matchDomainPattern(pattern, host) {
			for k, v := range headers {
				result[k] = v
			}
		}
	}

	return result
}

// extractHost extracts the hostname from a URL string
func extractHost(urlStr string) string {
	// Handle URLs without scheme
	if !strings.Contains(urlStr, "://") {
		urlStr = "https://" + urlStr
	}

	// Find the host portion
	start := strings.Index(urlStr, "://")
	if start == -1 {
		return ""
	}
	start += 3

	end := strings.IndexAny(urlStr[start:], ":/")
	if end == -1 {
		return urlStr[start:]
	}
	return urlStr[start : start+end]
}

// matchDomainPattern checks if a host matches a domain pattern.
// Supports wildcard prefix: "*.example.com" matches "www.example.com", "api.example.com"
// Exact match: "example.com" only matches "example.com"
func matchDomainPattern(pattern, host string) bool {
	pattern = strings.ToLower(pattern)
	host = strings.ToLower(host)

	// Wildcard pattern: *.example.com
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // ".example.com"
		// Match if host ends with the suffix (subdomain) or equals the base domain
		if strings.HasSuffix(host, suffix) {
			return true
		}
		// Also match the base domain itself (*.example.com should match example.com)
		baseDomain := pattern[2:]
		return host == baseDomain
	}

	// Exact match
	return pattern == host
}
