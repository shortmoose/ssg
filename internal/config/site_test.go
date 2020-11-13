package config_test

import (
	"reflect"
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
template: title-blah
unknown: unknown-blah`,
			cfg: config.Site{
				Template: "title-blah",
			},
		},
		{
			input: `
template: title-blah
custom:
  image: image-blah`,
			cfg: config.Site{
				Template: "title-blah",
				Custom:   map[string]interface{}{"image": "image-blah"},
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

		if !reflect.DeepEqual(tc.cfg, cfg) {
			t.Errorf("Expected = \n%#v, \nGot = \n%#v", tc.cfg, cfg)
		}
	}
}
