package funcs

import (
	"sort"

	"github.com/shortmoose/ssg/internal/config"
)

func SortPosts(configs []config.Post) []config.Post {
	ents := []config.Post{}
	for i := range configs {
		if configs[i].Date != "" {
			ents = append(ents, configs[i])
		}
	}
	sort.Sort(config.ByDate(ents))
	return ents
}

func FilterPosts(configs []config.Post, label string) []config.Post {
	ents := []config.Post{}
	for _, c := range configs {
		for _, l := range c.Labels {
			if l == label {
				ents = append(ents, c)
			}
		}
	}
	return ents
}

func ExcludePosts(configs []config.Post, label string) []config.Post {
	ents := []config.Post{}
	for _, c := range configs {
		found := false
		for _, l := range c.Labels {
			if l == label {
				found = true
			}
		}
		if !found {
			ents = append(ents, c)
		}
	}
	return ents
}

func TopNPosts(configs []config.Post, count int) []config.Post {
	if len(configs) > count {
		return configs[:count]
	}
	return configs
}
