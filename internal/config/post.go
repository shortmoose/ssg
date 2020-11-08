package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

// Post is the structure used for a given web post entry.
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

func stripPageConfig(body []byte) []byte {
	start := bytes.Index(body, []byte("<!-- CONFIG"))
	end := bytes.Index(body, []byte("CONFIG -->"))

	if start < 0 || end < 0 {
		return body
	}

	return append(body[:start], body[end+12:]...)
}

func getPageConfig(src string) (Post, error) {
	var cfg Post

	body, err := ioutil.ReadFile(src)
	if err != nil {
		return cfg, fmt.Errorf("reading file %s: %w", src, err)
	}

	start := bytes.Index(body, []byte("<!-- CONFIG"))
	end := bytes.Index(body, []byte("CONFIG -->"))

	if start >= 0 && end >= 0 {
		err = yaml.Unmarshal(body[start+11:end], &cfg)
		if err != nil {
			return cfg, fmt.Errorf("unmarshalling page config: %w", err)
		}
		body = stripPageConfig(body)
	}

	cfg.Content = body
	return cfg, nil
}

// GetPageConfig will parse the web post entry at filepath.
func GetPageConfig(filepath, sitepath string) (Post, error) {
	cfg, err := getPageConfig(filepath)
	if err != nil {
		return cfg, err
	}

	cfg.FilePath = filepath
	cfg.SitePath = sitepath

	if cfg.Date != "" {
		t, err := time.Parse(time.RFC3339, cfg.Date)
		if err != nil {
			return cfg, fmt.Errorf("parsing date %s: %w", cfg.Date, err)
		}
		date := t.Format(time.RFC3339)
		if cfg.Date != date {
			return cfg, fmt.Errorf("dates don't match: %s != %s", cfg.Date, date)
		}
	}

	return cfg, nil
}

// ByDate should normally be used as part of sort. So
// sort.Sort(ByDate([]Entry)).
type ByDate []Post

func (s ByDate) Len() int {
	return len(s)
}
func (s ByDate) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByDate) Less(i, j int) bool {
	return s[i].Date > s[j].Date
}
