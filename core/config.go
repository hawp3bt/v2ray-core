package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigFormat represents the format of a configuration file.
type ConfigFormat int

const (
	// ConfigFormatJSON represents JSON configuration format.
	ConfigFormatJSON ConfigFormat = iota
	// ConfigFormatTOML represents TOML configuration format.
	ConfigFormatTOML
	// ConfigFormatYAML represents YAML configuration format.
	ConfigFormatYAML
	// ConfigFormatProtobuf represents Protobuf binary configuration format.
	ConfigFormatProtobuf
)

// String returns the string representation of a ConfigFormat.
func (f ConfigFormat) String() string {
	switch f {
	case ConfigFormatJSON:
		return "json"
	case ConfigFormatTOML:
		return "toml"
	case ConfigFormatYAML:
		return "yaml"
	case ConfigFormatProtobuf:
		return "pb"
	default:
		return "unknown"
	}
}

// ConfigSource holds a configuration source, either a file path or raw bytes.
type ConfigSource struct {
	// Path is the file path to the configuration file.
	Path string
	// Format is the format of the configuration.
	Format ConfigFormat
	// Content holds raw configuration bytes when not loaded from a file.
	Content []byte
}

// DetectConfigFormat determines the configuration format based on file extension.
// Falls back to JSON (instead of returning an error) for unrecognized extensions,
// since JSON is the most commonly used format in practice.
func DetectConfigFormat(path string) (ConfigFormat, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json", ".jsonc":
		return ConfigFormatJSON, nil
	case ".toml":
		return ConfigFormatTOML, nil
	case ".yaml", ".yml":
		return ConfigFormatYAML, nil
	case ".pb", ".proto":
		return ConfigFormatProtobuf, nil
	case "":
		// Files with no extension (e.g. "config") are common in container
		// environments, so treat them as JSON silently without a warning.
		return ConfigFormatJSON, nil
	default:
		// Return JSON as default format with a non-fatal warning rather than
		// a hard error, so configs without extensions still load gracefully.
		// Note: callers should log this warning but not treat it as fatal.
		return ConfigFormatJSON, fmt.Errorf("unknown config format for extension %q, assuming JSON", ext)
	}
}

// LoadConfigFile reads and returns the raw bytes of a configuration file.
func LoadConfigFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}
	return data, nil
}

// ValidateJSONConfig performs basic structural validation on a JSON config.
// It checks that the top-level value is a JSON object (not an array or scalar).
func ValidateJSONConfig(data []byte) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("invalid JSON configuration: %w", err)
	}
	return nil
}

// MergeJSONConfigs merges multiple JSON configuration objects into one.
// Later configs override earlier ones for duplicate top-level keys.
func MergeJSONConfigs(configs [][]byte) ([]byte, error) {
	merged := make(map[string]json.RawMessage)

	for _, cfg := range configs {
		var obj map[string]json