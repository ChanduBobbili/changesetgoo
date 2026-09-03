package config

import (
    "encoding/json"
    "fmt"
    "os"
    "strings"

    "gopkg.in/yaml.v3"
)

type Config struct {
    ChangesDir        string `json:"changesDir" yaml:"changesDir"`
    TagPrefix         string `json:"tagPrefix" yaml:"tagPrefix"`
    CommitMessage     string `json:"commitMessage" yaml:"commitMessage"`
    ChangelogTemplate string `json:"changelogTemplate" yaml:"changelogTemplate"`
    IssueFormat       string `json:"issueFormat" yaml:"issueFormat"`
}

type partialConfig struct {
    ChangesDir        *string `json:"changesDir" yaml:"changesDir"`
    TagPrefix         *string `json:"tagPrefix" yaml:"tagPrefix"`
    CommitMessage     *string `json:"commitMessage" yaml:"commitMessage"`
    ChangelogTemplate *string `json:"changelogTemplate" yaml:"changelogTemplate"`
    IssueFormat       *string `json:"issueFormat" yaml:"issueFormat"`
}

var allowedKeys = map[string]struct{}{
    "changesDir":        {},
    "tagPrefix":         {},
    "commitMessage":     {},
    "changelogTemplate": {},
    "issueFormat":       {},
}

// Defaults mirrors current hardcoded behavior for zero-breaking adoption.
func Defaults() Config {
    return Config{
        ChangesDir:        ".changesets",
        TagPrefix:         "v",
        CommitMessage:     "chore 🚀: release {{tag}}",
        ChangelogTemplate: "## {{version}}",
        IssueFormat:       "#{{id}}",
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

    var parsed partialConfig
    if err := json.Unmarshal(data, &parsed); err != nil {
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

    var parsed partialConfig
    if err := yaml.Unmarshal(data, &parsed); err != nil {
        return partialConfig{}, nil, err
    }

    return parsed, unknown, nil
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
    if src.CommitMessage != nil {
        dst.CommitMessage = *src.CommitMessage
    }
    if src.ChangelogTemplate != nil {
        dst.ChangelogTemplate = *src.ChangelogTemplate
    }
    if src.IssueFormat != nil {
        dst.IssueFormat = *src.IssueFormat
    }
}

func validateConfig(cfg Config) {
    if !strings.Contains(cfg.CommitMessage, "{{tag}}") && !strings.Contains(cfg.CommitMessage, "{{version}}") {
        fmt.Fprintln(os.Stderr, "⚠️ commitMessage should include {{tag}} or {{version}}")
    }
    if !strings.Contains(cfg.ChangelogTemplate, "{{version}}") {
        fmt.Fprintln(os.Stderr, "⚠️ changelogTemplate should include {{version}}")
    }
    if !strings.Contains(cfg.IssueFormat, "{{id}}") {
        fmt.Fprintln(os.Stderr, "⚠️ issueFormat should include {{id}}")
    }
}