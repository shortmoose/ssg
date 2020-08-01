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

	"github.com/nthnca/ssg/internal/post"
	"github.com/nthnca/ssg/internal/util"
)

var (
	defaultTitle = "Web Site Title"
	defaultImage = "/img/logo.jpg"
	steURL       = "https://thisisntarealurl.com"
	author       = "Name"
)

// Config TODO
type feed struct {
	SiteURL   string
	SiteTitle string
	SiteID    string
	Author    string
}

func createAtomFeed(path string, feed feed, configs []post.Entry) error {
	cfgs := []post.Entry{}
	for i := range configs {
		if configs[i].Date != "" {
			cfgs = append(cfgs, configs[i])
		}
	}
	if len(cfgs) == 0 {
		return fmt.Errorf("Can't create XML feed, no entries")
	}
	sort.Sort(post.ByDate(cfgs))

	s := ""
	s += fmt.Sprintf("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n")
	s += fmt.Sprintf("<feed xmlns=\"http://www.w3.org/2005/Atom\">\n")
	s += fmt.Sprintf("  <title>%s</title>\n", feed.SiteTitle)
	s += fmt.Sprintf("  <link href=\"%s/\" />\n", feed.SiteURL)
	s += fmt.Sprintf("  <updated>%s</updated>\n", cfgs[0].Date)
	s += fmt.Sprintf("  <id>%s</id>\n", feed.SiteID)

	for _, e := range cfgs {
		if e.Date != "" {
			s += fmt.Sprintf("<entry>\n")
			s += fmt.Sprintf("  <title>%s</title>\n", e.Title)
			s += fmt.Sprintf("  <link href=\"%s%s\" />\n", feed.SiteURL, e.SitePath)
			s += fmt.Sprintf("  <updated>%s</updated>\n", e.Date)
			s += fmt.Sprintf("  <id>%s%s</id>\n", feed.SiteURL, e.SitePath)
			s += fmt.Sprintf("  <author><name>%s</name></author>\n", feed.Author)
			s += fmt.Sprintf("  <content type=\"html\"><![CDATA[\n")
			s += fmt.Sprintf("%s\n", e.Content)
			s += fmt.Sprintf("  ]]></content>\n")
			s += fmt.Sprintf("</entry>\n")
		}
	}
	s += fmt.Sprintf("</feed>\n")

	body := []byte(s)

	err := ioutil.WriteFile(path, body, 0644)
	if err != nil {
		return fmt.Errorf("WriteFile :%w", err)
	}

	return nil
}

func PostIndexEntry(e post.Entry) string {
	var cnt string
	img := e.Image
	if img == "" {
		img = defaultImage
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

func PostIndexEntryKey(key string, configs []post.Entry) (string, error) {
	for i := range configs {
		if configs[i].SitePath == key {
			return PostIndexEntry(configs[i]), nil
		}
	}

	return "", fmt.Errorf("invalid key: ''%s'", key)
}

func BuildIndex(path string, cfg post.Entry, configs []post.Entry) error {
	cfgs := []post.Entry{}
	for i := range configs {
		if configs[i].Date != "" {
			cfgs = append(cfgs, configs[i])
		}
	}
	if len(cfgs) == 0 {
		return fmt.Errorf("Can't create XML feed, no entries")
	}
	sort.Sort(post.ByDate(cfgs))

	cnt := ""
	for _, e := range cfgs {
		cnt += PostIndexEntry(e)
	}

	cfg.Content = []byte(cnt)
	err := BuildPage(path, cfg, configs)
	if err != nil {
		return err
	}

	return nil
}

func BuildPage(dest string, cfg post.Entry, configs []post.Entry) error {
	pre, err := ioutil.ReadFile("templates/pre.html")
	if err != nil {
		return err
	}

	pre = bytes.Replace(pre, []byte("<!--TITLE-->"), []byte(cfg.Title), -1)
	pre = bytes.Replace(pre, []byte("<!--DESCRIPTION-->"), []byte(cfg.Snippet), -1)
	pre = bytes.Replace(pre, []byte("<!--IMAGE-->"), []byte(cfg.Image), -1)

	meta := ""
	if cfg.Image != defaultImage {
		meta = "<meta property=\"og:image\" content=\"" + cfg.Image + "\" />\n  "
	}
	pre = bytes.Replace(pre, []byte("<!--META-->\n"), []byte(meta), -1)

	if cfg.Title != defaultTitle {
		pre = append(pre, []byte("<h1>"+cfg.Title+"</h1>\n")...)
	}

	body := cfg.Content

	var err_strings []string
	re := regexp.MustCompile(`<!--/.*-->`)
	body = re.ReplaceAllFunc(body, func(a []byte) []byte {
		key := string(a[4 : len(a)-3])
		html, err := PostIndexEntryKey(key, configs)
		if err != nil {
			err_strings = append(err_strings, key)
			return []byte("")
		}
		return []byte(html)
	})
	if len(err_strings) != 0 {
		return fmt.Errorf("Invalid keys: %v", err_strings)
	}

	var extra string
	for _, k := range cfg.RelatedPosts {
		html, err := PostIndexEntryKey(k, configs)
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

	err = ioutil.WriteFile(dest, body, 0644)
	if err != nil {
		return fmt.Errorf("WriteFile :%w", err)
	}

	return nil
}

func validateImagesExist(configs []post.Entry) error {
	m := map[string]bool{}
	for _, cfg := range configs {
		re := regexp.MustCompile(`/(img|pdf)/[^"']*`)
		for _, url := range re.FindAll([]byte(string(cfg.Content)+" "+cfg.Image), -1) {
			urlstr := "storageurl" + string(url)[4:]
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
	siteinfo.DefaultTitle = defaultTitle
	siteinfo.DefaultImage = defaultImage

	var configs []post.Entry
	err := util.Walk("posts", func(path string, info os.FileInfo) error {
		cfg, err := post.GetPageConfig(path, path[5:], siteinfo)
		if err != nil {
			return err
		}

		configs = append(configs, cfg)
		return nil
	})
	if err != nil {
		return err
	}

	for _, cfg := range configs {
		if cfg.Type == "index" {
			err := BuildIndex("website/posts"+cfg.SitePath, cfg, configs)
			if err != nil {
				return fmt.Errorf("parsing %s: %w", cfg.FilePath, err)
			}
		} else {
			err = BuildPage("website/posts"+cfg.SitePath, cfg, configs)
			if err != nil {
				return fmt.Errorf("parsing %s: %w", cfg.FilePath, err)
			}
		}
	}

	err = validateImagesExist(configs)
	if err != nil {
		return err
	}

	var feed feed
	feed.SiteTitle = defaultTitle
	feed.SiteURL = siteURL
	feed.SiteID = siteURL + "/"
	feed.Author = author

	err = createAtomFeed("website/atom.xml", feed, configs)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	err := walk()
	if err != nil {
		log.Fatalf("%v\n", err)
	}
}
