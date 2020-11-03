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
	"text/template"

	"github.com/shortmoose/ssg/internal/config"
	"github.com/shortmoose/ssg/internal/post"
	"github.com/shortmoose/ssg/internal/util"
)

var (
	cfg config.Config
)

func postIndexEntry(e post.Entry) string {
	var cnt string
	img := e.Image
	if img == "" {
		img = cfg.Image
	}
	cnt += "\n\n"
	cnt += "<div style=\"width: 100%; overflow: hidden;\">"
	cnt += "<div style=\"width: 170px; float: left;\">"
	cnt += "<a href=\"" + e.SitePath + "\">"
	cnt += "<img class=\"himg\" alt=\"thumbnail\" src=\"" + img + "\"></a></div>"
	cnt += "<div style=\"margin-left: 190px;\">"

	cnt += "<a href=\"" + e.SitePath + "\"><b>" + e.Title + "</b></a>"
	if e.Snippet != "" {
		cnt += "<p>" + e.Snippet + "</p>"
	}
	cnt += "</div></div><br />\n"

	return cnt
}

func postIndexEntryKey(key string, configs []post.Entry) (string, error) {
	for i := range configs {
		if configs[i].SitePath == key {
			return postIndexEntry(configs[i]), nil
		}
	}

	return "", fmt.Errorf("invalid key: ''%s'", key)
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

	cnt := ""
	for _, e := range ents {
		cnt += postIndexEntry(e)
	}

	ent.Content = []byte(cnt)
	err := buildPage(path, ent, configs)
	if err != nil {
		return err
	}

	return nil
}

func buildPage(dest string, ent post.Entry, configs []post.Entry) error {
	pre, err := ioutil.ReadFile("templates/pre.html")
	if err != nil {
		return err
	}

	pre = bytes.Replace(pre, []byte("<!--TITLE-->"), []byte(ent.Title), -1)
	pre = bytes.Replace(pre, []byte("<!--DESCRIPTION-->"), []byte(ent.Snippet), -1)
	pre = bytes.Replace(pre, []byte("<!--IMAGE-->"), []byte(ent.Image), -1)

	meta := ""
	if ent.Image != cfg.Image {
		meta = "<meta property=\"og:image\" content=\"" + ent.Image + "\" />\n  "
	}
	pre = bytes.Replace(pre, []byte("<!--META-->\n"), []byte(meta), -1)

	if ent.Title != cfg.Title {
		pre = append(pre, []byte("<h1>"+ent.Title+"</h1>\n")...)
	}

	body := ent.Content

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
		return fmt.Errorf("Invalid keys: %v", errStrings)
	}

	re2 := regexp.MustCompile(`<!--MACRO_WEB:.*-->`)
	body = re2.ReplaceAllFunc(body, func(a []byte) []byte {
		key := string(a[14 : len(a)-3])

		rv, err := expandMacro(key, ent.SitePath)
		if err != nil {
			errStrings = append(errStrings, err.Error())
			return []byte("")
		}

		return rv
	})
	if len(errStrings) != 0 {
		return fmt.Errorf("Invalid keys: %v", errStrings)
	}

	var extra string
	for _, k := range ent.RelatedPosts {
		html, err := postIndexEntryKey(k, configs)
		if err != nil {
			return err
		}
		extra += html
	}
	if len(extra) != 0 {
		extra = "\n<hr class=\"foo\">\n" +
			"<p><b>If you enjoyed that article, try out a couple more:</b></p>\n" +
			extra + "\n\n"
	}
	body = append(body, []byte(extra)...)

	post, err := ioutil.ReadFile("templates/post.html")
	if err != nil {
		return fmt.Errorf("ReadFile :%w", err)
	}
	body = append(pre, append(body, post...)...)

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

type Foo struct {
	UrlRelative string
}

func expandMacro(key, path string) ([]byte, error) {
	t, err := template.ParseFiles("templates/macros/" + key + ".tmpl")
	if err != nil {
		return nil, err
	}

	var f Foo
	f.UrlRelative = path
	out := new(bytes.Buffer)
	err = t.Execute(out, f)
	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
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

		var errStrings []string
		re2 := regexp.MustCompile(`<!--MACRO:.*-->`)
		ent.Content = re2.ReplaceAllFunc(ent.Content, func(a []byte) []byte {
			key := string(a[10 : len(a)-3])
			rv, err := expandMacro(key, ent.SitePath)
			if err != nil {
				errStrings = append(errStrings, err.Error())
				return []byte("")
			}

			return rv
		})
		if len(errStrings) != 0 {
			return fmt.Errorf("Invalid keys: %v", errStrings)
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
