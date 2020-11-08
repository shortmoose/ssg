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
	"github.com/shortmoose/ssg/internal/post"
	"github.com/shortmoose/ssg/internal/util"
)

var (
	cfg config.Config
)

func postIndexEntry(e post.Entry) ([]byte, error) {
	img := e.Image
	if img == "" {
		img = cfg.Image
	}

	var data util.PageData
	data.SiteConfig = cfg
	data.Entry = e

	return util.ExecuteTemplateByName("postlink", &data)
}

func postIndexEntryKey(key string, configs []post.Entry) ([]byte, error) {
	for i := range configs {
		if configs[i].SitePath == key {
			return postIndexEntry(configs[i])
		}
	}

	return nil, fmt.Errorf("invalid key: ''%s'", key)
}

func buildIndex(path string, ent post.Entry, configs []post.Entry) error {
	ents := []post.Entry{}
	for i := range configs {
		if configs[i].Date != "" {
			ents = append(ents, configs[i])
		}
	}
	if len(ents) == 0 {
		return fmt.Errorf("Can't create index, no entries")
	}
	sort.Sort(post.ByDate(ents))

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

func expandBody(ent post.Entry, configs []post.Entry, data util.PageData) ([]byte, error) {
	body, err := util.ExecuteTemplateGiven(string(ent.Content), data)
	if err != nil {
		return nil, err
	}

	var errStrings []string
	re := regexp.MustCompile(`<!--/.*-->`)
	body = re.ReplaceAllFunc(body, func(a []byte) []byte {
		key := string(a[4 : len(a)-3])
		html, err := postIndexEntryKey(key, configs)
		if err != nil {
			errStrings = append(errStrings, key)
			return []byte("")
		}
		return []byte(html)
	})
	if len(errStrings) != 0 {
		return nil, fmt.Errorf("Invalid keys: %v", errStrings)
	}

	var extra []byte
	for _, k := range ent.RelatedPosts {
		html, err := postIndexEntryKey(k, configs)
		if err != nil {
			return nil, err
		}
		extra = append(extra, html...)
	}

	ext := string(extra)
	if len(ext) != 0 {
		ext = "\n<hr class=\"foo\">\n" +
			"<p><b>If you enjoyed that article, try out a couple more:</b></p>\n" +
			ext + "\n\n"
	}
	body = append(body, []byte(ext)...)

	return body, nil
}

func buildPage(dest string, ent post.Entry, configs []post.Entry) error {
	var data util.PageData
	data.SiteConfig = cfg
	data.Entry = ent
	data.Web = true
	data.Pages = make(map[string]post.Entry)
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

func validateImagesExist(configs []post.Entry) error {
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
	var siteinfo post.SiteInfo
	siteinfo.DefaultTitle = cfg.Title
	siteinfo.DefaultImage = cfg.Image

	var configs []post.Entry
	err := util.Walk("posts", func(path string, info os.FileInfo) error {
		ent, err := post.GetPageConfig(path, path[5:], siteinfo)
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

	cfgTmp, err := config.GetConfig(handle)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	cfg = cfgTmp

	err = walk()
	if err != nil {
		log.Fatalf("%v\n", err)
	}
}
