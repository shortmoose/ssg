package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/shortmoose/ssg/internal/config"
	"github.com/shortmoose/ssg/internal/util"
)

var (
	cfg config.Site
)

// feed TODO
type Feed struct {
	SiteURL   string
	SiteTitle string
	SiteID    string
	Author    string
}

type PageData struct {
	config.Post

	Pages     map[string]config.Post
	PagesList []config.Post
	Body      string
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

func ExecuteTemplateGiven(templateText string, data PageData) ([]byte, error) {
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
			return cfg
		},
		"post": func() config.Post {
			log.Printf("Data Post: %v", data.Post.SitePath)
			return data.Post
		},
		"getPost": func(key string) config.Post {
			log.Printf("Data Post: %v", data.Post.SitePath)
			return data.Pages[key]
		},
		"allPosts": func() []config.Post {
			log.Printf("Data Post: %v", data.Post.SitePath)
			return data.PagesList
		},
		"sort2":   Sort,
		"recurse": recurse,
	}

	tmpl, err := template.New("x").Funcs(funcMap).Parse(templateText)
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
	err = tmpl.ExecuteTemplate(out, "x", data)
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

	bx, err := ExecuteTemplateGiven(string(ent.Content), data)
	if err != nil {
		return err
	}
	data.Body = string(bx)

	template := ent.Template
	if template == "" {
		template = cfg.Template
		if template == "" {
			log.Fatalf("No template name")
		}
	}

	body, err := ExecuteTemplateGiven("{{template \""+template+"\" .}}", data)
	if err != nil {
		return err
	}

	body = bytes.ReplaceAll(body, []byte("/img/"), []byte(cfg.ImageURL+"/"))
	body = bytes.ReplaceAll(body, []byte("/pdf/"), []byte(cfg.ImageURL+"/"))

	err = ioutil.WriteFile(dest, body, 0644)
	if err != nil {
		return fmt.Errorf("WriteFile :%w", err)
	}

	return nil
}

func validateImagesExist(configs []config.Post) error {
	m := map[string]bool{}
	for _, ent := range configs {
		re := regexp.MustCompile(`/(img|pdf)/[^"']*`)
		for _, url := range re.FindAll([]byte(string(ent.Content)+" "+ent.Image), -1) {
			urlstr := cfg.ImageURL + string(url)[4:]
			m[urlstr] = true
		}
	}

	for s := range m {
		res, err := http.Head(s)
		if err != nil {
			fmt.Printf("Error while looking for: %s\n", s)
			return err
		} else if res.StatusCode != 200 {
			fmt.Printf("Error: %s returned status %d\n", s, res.StatusCode)
			return fmt.Errorf("Blah")
		}
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

	err = validateImagesExist(configs)
	if err != nil {
		return err
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

	cfg = cfgTmp

	err = walk()
	if err != nil {
		log.Fatalf("%v\n", err)
	}
}
