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

	"github.com/shortmoose/ssg/internal/config"
	"github.com/shortmoose/ssg/internal/util"
)

var (
	cfg config.Site
)

func postIndexEntry(e config.Entry) ([]byte, error) {
	img := e.Image
	if img == "" {
		img = cfg.Image
	}

	var data util.PageData
	data.SiteConfig = cfg
	data.Entry = e

	return util.ExecuteTemplateByName("postlink", &data)
}

func buildIndex(path string, ent config.Entry, configs []config.Entry) error {
	ents := []config.Entry{}
	for i := range configs {
		if configs[i].Date != "" {
			ents = append(ents, configs[i])
		}
	}
	if len(ents) == 0 {
		return fmt.Errorf("Can't create index, no entries")
	}
	sort.Sort(config.ByDate(ents))

	var cnt []byte
	for _, e := range ents {
		x, err := postIndexEntry(e)
		if err != nil {
			return err
		}
		cnt = append(cnt, x...)
	}

	ent.Content = []byte(cnt)
	err := buildPage(path, ent, configs)
	if err != nil {
		return err
	}

	return nil
}

func expandBody(ent config.Entry, configs []config.Entry, data util.PageData) ([]byte, error) {
	body, err := util.ExecuteTemplateGiven(string(ent.Content), data)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func buildPage(dest string, ent config.Entry, configs []config.Entry) error {
	var data util.PageData
	data.SiteConfig = cfg
	data.Entry = ent
	data.Web = true
	data.Pages = make(map[string]config.Entry)
	for _, c := range configs {
		data.Pages[c.SitePath] = c
	}

	b, err := expandBody(ent, configs, data)
	data.Body = string(b)
	if err != nil {
		return err
	}

	body, err := util.ExecuteTemplateByName("pre", data)
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

func validateImagesExist(configs []config.Entry) error {
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
	var siteinfo config.SiteInfo
	siteinfo.DefaultTitle = cfg.Title
	siteinfo.DefaultImage = cfg.Image

	var configs []config.Entry
	err := util.Walk("posts", func(path string, info os.FileInfo) error {
		ent, err := config.GetPageConfig(path, path[5:], siteinfo)
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
		if ent.Type == "index" {
			err := buildIndex("website/posts"+ent.SitePath, ent, configs)
			if err != nil {
				return fmt.Errorf("parsing %s: %w", ent.FilePath, err)
			}
		} else if ent.Type == "atom" {
			var feed util.Feed
			feed.SiteTitle = cfg.Title
			feed.SiteURL = cfg.URL
			feed.SiteID = cfg.URL + "/"
			feed.Author = cfg.Author

			body, err := util.CreateAtomFeed(feed, configs)
			if err != nil {
				return err
			}

			body = bytes.ReplaceAll(body, []byte("/img/"), []byte(cfg.ImageURL+"/"))
			body = bytes.ReplaceAll(body, []byte("/pdf/"), []byte(cfg.ImageURL+"/"))
			body = bytes.ReplaceAll(body, []byte("href=\"/"), []byte("href=\""+feed.SiteURL+"/"))

			err = ioutil.WriteFile("website/posts"+ent.SitePath, body, 0644)
			if err != nil {
				return fmt.Errorf("WriteFile :%w", err)
			}
		} else {
			err = buildPage("website/posts"+ent.SitePath, ent, configs)
			if err != nil {
				return fmt.Errorf("parsing %s: %w", ent.FilePath, err)
			}
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
