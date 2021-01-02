package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/shortmoose/ssg/internal/config"
	"github.com/shortmoose/ssg/internal/util"
)

var (
	siteConfig config.Site
	postList   []config.Post
	postMap    map[string]config.Post
)

func sortPosts(configs []config.Post) []config.Post {
	ents := []config.Post{}
	for i := range configs {
		if configs[i].Date != "" {
			ents = append(ents, configs[i])
		}
	}
	sort.Sort(config.ByDate(ents))
	return ents
}

func filterPosts(configs []config.Post, label string) []config.Post {
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

func topNPosts(configs []config.Post, count int) []config.Post {
	if len(configs) > count {
		return configs[:count]
	}
	return configs
}

func recurse(tmpl *template.Template, name string, data interface{}) (string, error) {
	_, err := tmpl.New("y").Parse(name)
	if err != nil {
		return "", fmt.Errorf("parsing recursive template: %w", err)
	}

	out := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(out, "y", data)
	if err != nil {
		return "", fmt.Errorf("executing recursive template: %w", err)
	}

	return string(out.Bytes()), err
}

func createFuncMap(post config.Post, tmpl **template.Template) template.FuncMap {
	return template.FuncMap{
		"siteConfig": func() config.Site {
			return siteConfig
		},
		"post": func() config.Post {
			return post
		},
		"getPost": func(key string) config.Post {
			return postMap[key]
		},
		"allPosts": func() []config.Post {
			return postList
		},
		"sort":   sortPosts,
		"filter": filterPosts,
		"topN":   topNPosts,
		"recurse": func(text string, data interface{}) (string, error) {
			return recurse(*tmpl, text, data)
		},
	}
}

func executeTemplate(templateName, templateText string, ent config.Post) ([]byte, error) {
	var gtmp *template.Template
	funcMap := createFuncMap(ent, &gtmp)
	tmpl, err := template.New("body").Option("missingkey=error").
		Funcs(funcMap).Parse(templateText)
	if err != nil {
		return nil, fmt.Errorf("parsing template '%v': %w", ent.FilePath, err)
	}

	_, err = tmpl.ParseGlob("templates/*")
	if err != nil {
		// TODO: There must be a better way to do this.
		if !strings.Contains(err.Error(), "matches no files") {
			return nil, fmt.Errorf("parsing templates/*: %w", err)
		}
	}
	gtmp = tmpl

	out := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(out, templateName, ent)
	if err != nil {
		return nil, fmt.Errorf("executing template '%v': %w", ent.FilePath, err)
	}

	return out.Bytes(), err
}

func buildPage(dest string, ent config.Post) error {
	template := ent.Template
	if template == "" {
		template = siteConfig.Template
		if template == "" {
			return fmt.Errorf("no template name '%v'", ent.FilePath)
		}
	}

	body, err := executeTemplate(template, string(ent.Content), ent)
	if err != nil {
		return fmt.Errorf("executing template '%v': %w", ent.FilePath, err)
	}

	err = ioutil.WriteFile(dest, body, 0644)
	if err != nil {
		return fmt.Errorf("writing file '%v': %w", dest, err)
	}

	return nil
}

func walk() error {
	err := util.Walk("posts", func(path string, info os.FileInfo) error {
		ent, err := config.GetPageConfig(path, path[5:])
		if err != nil {
			return fmt.Errorf("getting page config '%v': %w", path, err)
		}

		postList = append(postList, ent)
		return nil
	})
	if err != nil {
		return err
	}

	postMap = make(map[string]config.Post)
	for _, c := range postList {
		postMap[c.SitePath] = c
	}

	for _, ent := range postList {
		err = buildPage("website/posts"+ent.SitePath, ent)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", ent.FilePath, err)
		}
	}

	return nil
}

func main() {
	handle, err := os.Open("ssg.yaml")
	if err != nil {
		log.Fatalf("%v\n", err)
	}
	defer handle.Close()

	cfgTmp, err := config.GetSiteConfig(handle)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	siteConfig = cfgTmp

	err = walk()
	if err != nil {
		log.Fatalf("%+v", err)
	}
}
