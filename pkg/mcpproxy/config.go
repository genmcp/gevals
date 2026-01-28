package mcpproxy

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/genmcp/gen-mcp/pkg/template"
	"github.com/genmcp/gevals/pkg/util"
	"sigs.k8s.io/yaml"
)

const (
	TransportTypeHttp        = "http"
	TransportTypeStdio       = "stdio"
	TransportTypeManagedHttp = "managedhttp"
)

// MCPConfig represents the top-level MCP configuration file structure
// used by Claude Code, Cursor, and other MCP clients.
type MCPConfig struct {
	MCPServers map[string]*ServerConfig `json:"mcpServers" yaml:"mcpServers"`
}

// ServerConfig represents the configuration for a single MCP server.
// Supports both stdio (command-based) and HTTP-based servers.
type ServerConfig struct {
	// Type specifies the server type: "stdio", "managedhttp", or "http"
	// If not specified, will be inferred from URL (http) or Command (stdio)
	Type string `json:"type,omitempty"`

	// ServerUrlTemplate specifies what the server's URL will be once it's running.
	// It can use variables from 'command.env' or the 'lifecycle.setup' script output.
	// variables are formatted as ${ENV_VAR_NAME}
	ServerUrlTemplate string `json:"serverUrlTemplate,omitempty"`

	// ParsedServerUrlTemplate contains the parsed template for the server url
	ParsedServerUrlTemplate *template.ParsedTemplate `json:"-"`

	// Command is the executable to run (e.g., "node", "python", "npx")
	// Used for stdio and managedhttp servers
	Command string `json:"command,omitempty"`

	// Args are the command-line arguments to pass to the command
	// Used for stdio and managedhttp servers
	// Can contain env variables formatted as ${ENV_VAR_NAME}
	Args []string `json:"args,omitempty"`

	// ParsedArgs contains the parsed templates for the args
	ParsedArgs []*template.ParsedTemplate `json:"-"`

	// Env contains environment variables to set for the server process
	// Used for stdio and managedhttp servers
	Env map[string]string `json:"env,omitempty"`

	// HealthCheckConfig contains the config to health check the managedhttp
	// or http mcp servers, before accessing them
	HealthCheckConfig *HttpHealthCheckConfig `json:"healthCheck,omitempty"`

	// URL is the HTTP endpoint for the MCP server
	// Used for http servers. May contain environment variable references
	// like ${VAR} or ${VAR:-default}
	URL string `json:"url,omitempty"`

	// Headers are HTTP headers to send with requests
	// Used for http servers. Values may contain environment variable references
	Headers map[string]string `json:"headers,omitempty"`

	// Disabled indicates whether this server should be skipped
	Disabled bool `json:"disabled,omitempty"`

	// AlwaysAllow is a list of tools/resources that should always be allowed
	AlwaysAllow []string `json:"alwaysAllow,omitempty"`

	// EnableAllTools sets all tools to be allowed
	EnableAllTools bool `json:"enableAllTools"`

	// Lifecycle contains the lifecycle for the mcp server/env
	Lifecycle *McpServerLifecycleConfig `json:"lifecycle,omitempty"`
}

type HttpHealthCheckConfig struct {
	Path     string        `json:"path"`
	Retries  int           `json:"retries"`
	Interval time.Duration `json:"interval"`
}

type McpServerLifecycleConfig struct {
	// Setup contains setup script for the mcp server/env
	// Output that matches the format VAR=VALUE will be added to future env
	Setup *util.Step `json:"setup,omitempty"`
	// Cleanup contains cleanup script for the mcp server/env
	Cleanup *util.Step `json:"cleanup,omitempty"`
}

func (c *HttpHealthCheckConfig) UnmarshalJSON(data []byte) error {
	type Doppleganger HttpHealthCheckConfig

	tmp := &struct {
		Interval string `json:"interval"`
		*Doppleganger
	}{
		Doppleganger: (*Doppleganger)(c),
	}

	err := json.Unmarshal(data, tmp)
	if err != nil {
		return err
	}

	c.Interval, err = time.ParseDuration(tmp.Interval)
	if err != nil {
		return fmt.Errorf("unable to parse interval: %w", err)
	}

	return nil
}

func (c *ServerConfig) UnmarshalJSON(data []byte) error {
	type Doppleganger ServerConfig

	tmp := (*Doppleganger)(c)

	err := json.Unmarshal(data, tmp)
	if err != nil {
		return err
	}

	c.ParsedServerUrlTemplate, err = template.ParseTemplate(c.ServerUrlTemplate, template.TemplateParserOptions{})
	if err != nil {
		return fmt.Errorf("failed to parse serverUrlTemplate: %w", err)
	}

	c.ParsedArgs = make([]*template.ParsedTemplate, len(c.Args))

	for i, arg := range c.Args {
		c.ParsedArgs[i], err = template.ParseTemplate(arg, template.TemplateParserOptions{})
		if err != nil {
			return fmt.Errorf("failed to parse args[%d]: %w", i, err)
		}
	}

	return nil
}

// ParseConfigFile reads and parses an MCP config file from the given path.
// The file can be in JSON or YAML format.
func ParseConfigFile(path string) (*MCPConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return ParseConfig(data)
}

// ParseConfig parses MCP config data from bytes.
// The data can be in JSON or YAML format.
func ParseConfig(data []byte) (*MCPConfig, error) {
	var config MCPConfig

	// sigs.k8s.io/yaml can handle both JSON and YAML
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// validateConfig validates the parsed configuration.
func validateConfig(config *MCPConfig) error {
	if config.MCPServers == nil {
		return fmt.Errorf("mcpServers field is required")
	}

	for name, server := range config.MCPServers {
		if server.IsHttp() {
			if server.URL == "" {
				return fmt.Errorf("server %q: url is required for http servers", name)
			}
		} else if server.IsStdio() {
			if server.Command == "" {
				return fmt.Errorf("server %q: command is required for stdio servers", name)
			}
		} else {
			return fmt.Errorf("server %q: must specify either command or url", name)
		}
	}

	return nil
}

// GetEnabledServers returns a map of server names to their configurations,
// excluding any servers marked as disabled.
func (c *MCPConfig) GetEnabledServers() map[string]*ServerConfig {
	enabled := make(map[string]*ServerConfig)
	for name, server := range c.MCPServers {
		if !server.Disabled {
			enabled[name] = server
		}
	}
	return enabled
}

// GetServer returns the configuration for a specific server by name.
func (c *MCPConfig) GetServer(name string) (*ServerConfig, bool) {
	server, ok := c.MCPServers[name]
	return server, ok
}

// ToFile writes the configuration to the specified path
func (c *MCPConfig) ToFile(path string) error {
	bytes, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal MCPConfig to bytes: %w", err)
	}

	err = os.WriteFile(path, bytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write MCPConfig to file at path '%s': %w", path, err)
	}

	return nil
}

// IsStdio returns true if this is a stdio-based (command) server.
func (s *ServerConfig) IsStdio() bool {
	if s.Type == "stdio" {
		return true
	}
	if s.Type == "http" {
		return false
	}
	// Type not specified - infer from fields
	return s.Command != ""
}

// IsHttp returns true if this is an HTTP-based server.
func (s *ServerConfig) IsHttp() bool {
	if s.Type == "http" {
		return true
	}
	if s.Type == "stdio" {
		return false
	}
	// Type not specified - infer from fields
	return s.URL != ""
}
