package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ChangesDir          string          `json:"changesDir" yaml:"changesDir"`
	TagPrefix           string          `json:"tagPrefix" yaml:"tagPrefix"`
	IssueFormat         string          `json:"issueFormat" yaml:"issueFormat"`
	Changelog           ChangelogConfig `json:"changelog" yaml:"changelog"`
	Commit              CommitConfig    `json:"commit" yaml:"commit"`
	BaseBranch          string          `json:"baseBranch" yaml:"baseBranch"`
	ChangedFilePatterns []string        `json:"changedFilePatterns" yaml:"changedFilePatterns"`
	Snapshot            SnapshotConfig  `json:"snapshot" yaml:"snapshot"`
}

type ChangelogConfig struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	Template string `json:"template" yaml:"template"`
}

type CommitConfig struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Message string `json:"message" yaml:"message"`
}

type SnapshotConfig struct {
	UseCalculatedVersion bool   `json:"useCalculatedVersion" yaml:"useCalculatedVersion"`
	PrereleaseTemplate   string `json:"prereleaseTemplate" yaml:"prereleaseTemplate"`
}

type partialConfig struct {
	ChangesDir             *string
	TagPrefix              *string
	IssueFormat            *string
	Changelog              *ChangelogConfig
	Commit                 *CommitConfig
	BaseBranch             *string
	ChangedFilePatterns    []string
	HasChangedFilePatterns bool
	Snapshot               *SnapshotConfig
}

var allowedKeys = map[string]struct{}{
	"changesDir":          {},
	"tagPrefix":           {},
	"issueFormat":         {},
	"changelog":           {},
	"commit":              {},
	"baseBranch":          {},
	"changedFilePatterns": {},
	"snapshot":            {},
}

// Defaults mirrors current hardcoded behavior for zero-breaking adoption.
func Defaults() Config {
	return Config{
		ChangesDir:  ".changesets",
		TagPrefix:   "v",
		IssueFormat: "#{{id}}",
		Changelog: ChangelogConfig{
			Enabled:  true,
			Template: "## {{version}}",
		},
		Commit: CommitConfig{
			Enabled: true,
			Message: "chore 🚀: release {{tag}}",
		},
		BaseBranch:          "main",
		ChangedFilePatterns: []string{"**"},
		Snapshot: SnapshotConfig{
			UseCalculatedVersion: false,
			PrereleaseTemplate:   "{tag}-{datetime}",
		},
	}
}

// candidateFiles in precedence order.
var candidateFiles = []string{".changesetgoorc", "changesetgoo.json", "changesetgoo.yaml", "changesetgoo.yml"}

// LoadConfig searches the repo root for a config file, parses it, and merges
// it with defaults. If no config file is found, defaults are returned.
func LoadConfig() (Config, error) {
	cfg := Defaults()

	for _, filename := range candidateFiles {
		if _, err := os.Stat(filename); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return cfg, fmt.Errorf("failed to access %s: %w", filename, err)
		}

		data, err := os.ReadFile(filename)
		if err != nil {
			return cfg, fmt.Errorf("failed to read %s: %w", filename, err)
		}

		partial, unknown, err := decodeConfig(filename, data)
		if err != nil {
			return cfg, fmt.Errorf("failed to parse %s: %w", filename, err)
		}

		for _, key := range unknown {
			fmt.Fprintf(os.Stderr, "⚠️ Unknown config key '%s' in %s\n", key, filename)
		}

		mergeConfig(&cfg, partial)
		validateConfig(cfg)
		return cfg, nil
	}

	return cfg, nil
}

func decodeConfig(filename string, data []byte) (partialConfig, []string, error) {
	if filename == ".changesetgoorc" {
		trimmed := strings.TrimSpace(string(data))
		if strings.HasPrefix(trimmed, "{") {
			return decodeJSONConfig(data)
		}
		return decodeYAMLConfig(data)
	}

	if strings.HasSuffix(filename, ".json") {
		return decodeJSONConfig(data)
	}

	return decodeYAMLConfig(data)
}

func decodeJSONConfig(data []byte) (partialConfig, []string, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return partialConfig{}, nil, err
	}

	unknown := findUnknownKeys(raw)
	parsed, err := parsePartialFromRaw(raw)
	if err != nil {
		return partialConfig{}, nil, err
	}

	return parsed, unknown, nil
}

func decodeYAMLConfig(data []byte) (partialConfig, []string, error) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return partialConfig{}, nil, err
	}

	unknown := findUnknownKeys(raw)
	parsed, err := parsePartialFromRaw(raw)
	if err != nil {
		return partialConfig{}, nil, err
	}

	return parsed, unknown, nil
}

func parsePartialFromRaw(raw map[string]interface{}) (partialConfig, error) {
	var partial partialConfig

	if value, ok := raw["changesDir"]; ok {
		parsed, ok := value.(string)
		if !ok {
			return partial, fmt.Errorf("changesDir must be a string")
		}
		partial.ChangesDir = &parsed
	}

	if value, ok := raw["tagPrefix"]; ok {
		parsed, ok := value.(string)
		if !ok {
			return partial, fmt.Errorf("tagPrefix must be a string")
		}
		partial.TagPrefix = &parsed
	}

	if value, ok := raw["issueFormat"]; ok {
		parsed, ok := value.(string)
		if !ok {
			return partial, fmt.Errorf("issueFormat must be a string")
		}
		partial.IssueFormat = &parsed
	}

	if value, ok := raw["changelog"]; ok {
		parsed, err := parseChangelogConfig(value)
		if err != nil {
			return partial, err
		}
		partial.Changelog = &parsed
	}

	if value, ok := raw["commit"]; ok {
		parsed, err := parseCommitConfig(value)
		if err != nil {
			return partial, err
		}
		partial.Commit = &parsed
	}

	if value, ok := raw["baseBranch"]; ok {
		parsed, ok := value.(string)
		if !ok {
			return partial, fmt.Errorf("baseBranch must be a string")
		}
		partial.BaseBranch = &parsed
	}

	if value, ok := raw["changedFilePatterns"]; ok {
		items, ok := value.([]interface{})
		if !ok {
			return partial, fmt.Errorf("changedFilePatterns must be an array of strings")
		}
		patterns := make([]string, 0, len(items))
		for _, item := range items {
			str, ok := item.(string)
			if !ok {
				return partial, fmt.Errorf("changedFilePatterns must contain only strings")
			}
			patterns = append(patterns, str)
		}
		partial.ChangedFilePatterns = patterns
		partial.HasChangedFilePatterns = true
	}

	if value, ok := raw["snapshot"]; ok {
		parsed, err := parseSnapshotConfig(value)
		if err != nil {
			return partial, err
		}
		partial.Snapshot = &parsed
	}

	return partial, nil
}

func parseChangelogConfig(value interface{}) (ChangelogConfig, error) {
	result := Defaults().Changelog

	switch typed := value.(type) {
	case bool:
		result.Enabled = typed
		return result, nil
	case string:
		result.Template = typed
		return result, nil
	case map[string]interface{}:
		if enabled, ok := typed["enabled"]; ok {
			parsed, ok := enabled.(bool)
			if !ok {
				return result, fmt.Errorf("changelog.enabled must be a boolean")
			}
			result.Enabled = parsed
		}
		if template, ok := typed["template"]; ok {
			parsed, ok := template.(string)
			if !ok {
				return result, fmt.Errorf("changelog.template must be a string")
			}
			result.Template = parsed
		}
		return result, nil
	default:
		return result, fmt.Errorf("changelog must be a boolean, a template string, or an object with enabled/template")
	}
}

func parseCommitConfig(value interface{}) (CommitConfig, error) {
	result := Defaults().Commit

	switch typed := value.(type) {
	case bool:
		result.Enabled = typed
		return result, nil
	case string:
		result.Enabled = true
		result.Message = typed
		return result, nil
	case map[string]interface{}:
		if enabled, ok := typed["enabled"]; ok {
			parsed, ok := enabled.(bool)
			if !ok {
				return result, fmt.Errorf("commit.enabled must be a boolean")
			}
			result.Enabled = parsed
		}
		if message, ok := typed["message"]; ok {
			parsed, ok := message.(string)
			if !ok {
				return result, fmt.Errorf("commit.message must be a string")
			}
			result.Message = parsed
		}
		return result, nil
	default:
		return result, fmt.Errorf("commit must be a boolean, a message template string, or an object with enabled/message")
	}
}

func parseSnapshotConfig(value interface{}) (SnapshotConfig, error) {
	result := Defaults().Snapshot
	mapValue, ok := value.(map[string]interface{})
	if !ok {
		return result, fmt.Errorf("snapshot must be an object")
	}

	if useCalculated, ok := mapValue["useCalculatedVersion"]; ok {
		parsed, ok := useCalculated.(bool)
		if !ok {
			return result, fmt.Errorf("snapshot.useCalculatedVersion must be a boolean")
		}
		result.UseCalculatedVersion = parsed
	}

	if template, ok := mapValue["prereleaseTemplate"]; ok {
		parsed, ok := template.(string)
		if !ok {
			return result, fmt.Errorf("snapshot.prereleaseTemplate must be a string")
		}
		result.PrereleaseTemplate = parsed
	}

	return result, nil
}

func findUnknownKeys(raw map[string]interface{}) []string {
	unknown := make([]string, 0)
	for key := range raw {
		if _, ok := allowedKeys[key]; !ok {
			unknown = append(unknown, key)
		}
	}
	return unknown
}

func mergeConfig(dst *Config, src partialConfig) {
	if src.ChangesDir != nil {
		dst.ChangesDir = *src.ChangesDir
	}
	if src.TagPrefix != nil {
		dst.TagPrefix = *src.TagPrefix
	}
	if src.IssueFormat != nil {
		dst.IssueFormat = *src.IssueFormat
	}
	if src.Changelog != nil {
		dst.Changelog = *src.Changelog
	}
	if src.Commit != nil {
		dst.Commit = *src.Commit
	}
	if src.BaseBranch != nil {
		dst.BaseBranch = *src.BaseBranch
	}
	if src.HasChangedFilePatterns {
		dst.ChangedFilePatterns = src.ChangedFilePatterns
	}
	if src.Snapshot != nil {
		dst.Snapshot.UseCalculatedVersion = src.Snapshot.UseCalculatedVersion
		if src.Snapshot.PrereleaseTemplate != "" {
			dst.Snapshot.PrereleaseTemplate = src.Snapshot.PrereleaseTemplate
		}
	}
}

func validateConfig(cfg Config) {
	if !strings.Contains(cfg.Commit.Message, "{{tag}}") && !strings.Contains(cfg.Commit.Message, "{{version}}") {
		fmt.Fprintln(os.Stderr, "⚠️ commit.message should include {{tag}} or {{version}}")
	}
	if !strings.Contains(cfg.Changelog.Template, "{{version}}") {
		fmt.Fprintln(os.Stderr, "⚠️ changelog.template should include {{version}}")
	}
	if !strings.Contains(cfg.IssueFormat, "{{id}}") {
		fmt.Fprintln(os.Stderr, "⚠️ issueFormat should include {{id}}")
	}
	if strings.TrimSpace(cfg.BaseBranch) == "" {
		fmt.Fprintln(os.Stderr, "⚠️ baseBranch is empty; default branch comparisons may not work")
	}
	if cfg.Snapshot.PrereleaseTemplate != "" {
		hasKnownToken := strings.Contains(cfg.Snapshot.PrereleaseTemplate, "{tag}") ||
			strings.Contains(cfg.Snapshot.PrereleaseTemplate, "{commit}") ||
			strings.Contains(cfg.Snapshot.PrereleaseTemplate, "{commit-short}") ||
			strings.Contains(cfg.Snapshot.PrereleaseTemplate, "{timestamp}") ||
			strings.Contains(cfg.Snapshot.PrereleaseTemplate, "{datetime}")
		if !hasKnownToken {
			fmt.Fprintln(os.Stderr, "⚠️ snapshot.prereleaseTemplate does not include known placeholders")
		}
	}
}
