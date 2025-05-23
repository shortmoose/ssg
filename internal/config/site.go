package config

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// Entry is the structure used for a given web post entry.
type Site struct {
	Template string                 `yaml:"template"`
	Custom   map[string]interface{} `yaml:"custom"`
}

func GetSiteConfig(src io.Reader) (Site, error) {
	var cfg Site

	body, err := io.ReadAll(src)
	if err != nil {
		return cfg, fmt.Errorf("reading file %s: %w", src, err)
	}

	err = yaml.Unmarshal(body, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("unmarshalling '%s': %w", src, err)
	}
	return cfg, nil
}
