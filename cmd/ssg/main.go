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
)

type PageData struct {
	config.Post

	Pages     map[string]config.Post
	PagesList []config.Post
}

func Sort(configs []config.Post) []config.Post {
	ents := []config.Post{}
	for i := range configs {
		if configs[i].Date != "" {
			ents = append(ents, configs[i])
		}
	}
	sort.Sort(config.ByDate(ents))
	return ents
}

func ExecuteTemplateGiven(templateName, templateText string, data PageData) ([]byte, error) {
	var gtmp *template.Template
	recurse := func(name string, ydata interface{}) (string, error) {
		_, err := gtmp.New("y").Parse(name)
		if err != nil {
			log.Fatalf("OOPS")
		}

		nout := new(bytes.Buffer)
		err = gtmp.ExecuteTemplate(nout, "y", ydata)
		if err != nil {
			return "", err
		}

		return string(nout.Bytes()), err
	}

	funcMap := template.FuncMap{
		"siteConfig": func() config.Site {
			return siteConfig
		},
		"post": func() config.Post {
			return data.Post
		},
		"getPost": func(key string) config.Post {
			return data.Pages[key]
		},
		"allPosts": func() []config.Post {
			return data.PagesList
		},
		"sort":    Sort,
		"recurse": recurse,
	}

	tmpl, err := template.New("body").Funcs(funcMap).Parse(templateText)
	if err != nil {
		return nil, err
	}

	_, err = tmpl.ParseGlob("templates/*")
	if err != nil {
		// TODO: There must be a better way to do this.
		if !strings.Contains(err.Error(), "matches no files") {
			log.Printf("%v", err)
			return nil, err
		}
	}
	gtmp = tmpl

	out := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(out, templateName, data.Post)
	if err != nil {
		return nil, err
	}

	return out.Bytes(), err
}

func buildPage(dest string, ent config.Post, configs []config.Post) error {
	var data PageData
	data.Post = ent
	data.Pages = make(map[string]config.Post)
	data.PagesList = configs
	for _, c := range configs {
		data.Pages[c.SitePath] = c
	}

	template := ent.Template
	if template == "" {
		template = siteConfig.Template
		if template == "" {
			log.Fatalf("No template name")
		}
	}

	body, err := ExecuteTemplateGiven(template, string(ent.Content), data)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dest, body, 0644)
	if err != nil {
		return fmt.Errorf("WriteFile :%w", err)
	}

	return nil
}

func walk() error {
	var configs []config.Post
	err := util.Walk("posts", func(path string, info os.FileInfo) error {
		ent, err := config.GetPageConfig(path, path[5:])
		if err != nil {
			return err
		}

		configs = append(configs, ent)
		return nil
	})
	if err != nil {
		return err
	}

	for _, ent := range configs {
		err = buildPage("website/posts"+ent.SitePath, ent, configs)
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
		log.Fatalf("%v\n", err)
	}
}
