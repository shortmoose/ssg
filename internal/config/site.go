package config

import (
	"fmt"
	"io"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Entry is the structure used for a given web post entry.
type Site struct {
	Title    string `yaml:"title"`
	Image    string `yaml:"image"`
	URL      string `yaml:"url"`
	Author   string `yaml:"author"`
	ImageURL string `yaml:"image-url"`
}

type Post struct {
	Type         string   `yaml:"type"`
	Title        string   `yaml:"title"`
	Snippet      string   `yaml:"snippet"`
	Image        string   `yaml:"image"`
	Date         string   `yaml:"date"`
	RelatedPosts []string `yaml:"related"`
	Author       string   `yaml:"author"`

	FilePath string
	SitePath string
	Content  []byte
}

func GetSiteConfig(src io.Reader) (Site, error) {
	var cfg Site

	body, err := ioutil.ReadAll(src)
	if err != nil {
		return cfg, fmt.Errorf("reading file %s: %w", src, err)
	}

	err = yaml.Unmarshal(body, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("unmarshalling '%s': %w", src, err)
	}
	return cfg, nil
}
