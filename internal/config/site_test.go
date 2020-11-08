package config_test

import (
	"strings"
	"testing"

	"github.com/shortmoose/ssg/internal/config"
)

func TestGetSiteConfig(t *testing.T) {
	type test struct {
		input string
		cfg   config.Site
		err   string
	}

	tests := []test{
		{
			input: "Fancy Feet",
			cfg:   config.Site{},
			err:   "cannot unmarshal",
		},
		{
			input: `
title: title-blah
unknown: unknown-blah`,
			cfg: config.Site{
				Title: "title-blah",
			},
		},
		{
			input: `
title: title-blah
image: image-blah
url: url-blah
author: author-blah
image-url: image-url-blah`,
			cfg: config.Site{
				Title:    "title-blah",
				Image:    "image-blah",
				URL:      "url-blah",
				Author:   "author-blah",
				ImageURL: "image-url-blah",
			},
		},
	}

	for _, tc := range tests {
		cfg, err := config.GetSiteConfig(strings.NewReader(tc.input))
		if err != nil {
			if !strings.Contains(err.Error(), tc.err) {
				t.Errorf("Error = %v", err)
			}
			continue
		}

		if tc.err != "" {
			t.Errorf("Expected an error of %v", tc.err)
			continue
		}

		if tc.cfg != cfg {
			t.Errorf("Expected = \n%#v, \nGot = \n%#v", tc.cfg, cfg)
		}
	}
}
