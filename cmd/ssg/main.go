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

	SiteConfig config.Site
	Pages      map[string]config.Post
	PagesList  []config.Post
	Body       string
	Web        bool
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

func ExecuteTemplateByName(templateName string, data interface{}) ([]byte, error) {
	t, err := template.ParseGlob("templates/*")
	if err != nil {
		return nil, err
	}

	out := new(bytes.Buffer)
	err = t.ExecuteTemplate(out, templateName, data)
	if err != nil {
		log.Fatalf("Oops %s %v", templateName, err.Error())
	}
	return out.Bytes(), err
}

func ExecuteTemplateGiven(templateText string, data interface{}) ([]byte, error) {
	funcMap := template.FuncMap{
		"sort2": Sort,
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

	out := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(out, "x", data)
	if err != nil {
		return nil, err
	}

	return out.Bytes(), err
}

func CreateAtomFeed(feed Feed, configs []config.Post) ([]byte, error) {
	ents := []config.Post{}
	for i := range configs {
		if configs[i].Date != "" {
			ents = append(ents, configs[i])
		}
	}
	if len(ents) == 0 {
		return nil, fmt.Errorf("Can't create XML feed, no entries")
	}
	sort.Sort(config.ByDate(ents))

	s := ""
	s += fmt.Sprintf("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n")
	s += fmt.Sprintf("<feed xmlns=\"http://www.w3.org/2005/Atom\">\n")
	s += fmt.Sprintf("  <title>%s</title>\n", feed.SiteTitle)
	s += fmt.Sprintf("  <link href=\"%s/\" />\n", feed.SiteURL)
	s += fmt.Sprintf("  <updated>%s</updated>\n", ents[0].Date)
	s += fmt.Sprintf("  <id>%s</id>\n", feed.SiteID)

	for _, e := range ents {
		if e.Date != "" {
			s += fmt.Sprintf("<entry>\n")
			s += fmt.Sprintf("  <title>%s</title>\n", e.Title)
			s += fmt.Sprintf("  <link href=\"%s%s\" />\n", feed.SiteURL, e.SitePath)
			s += fmt.Sprintf("  <updated>%s</updated>\n", e.Date)
			s += fmt.Sprintf("  <id>%s%s</id>\n", feed.SiteURL, e.SitePath)
			s += fmt.Sprintf("  <author><name>%s</name></author>\n", feed.Author)
			s += fmt.Sprintf("  <content type=\"html\"><![CDATA[\n")

			var data PageData
			data.Post = e
			data.Pages = make(map[string]config.Post)

			c, err := ExecuteTemplateGiven(string(e.Content), data)
			if err != nil {
				return nil, err
			}

			s += fmt.Sprintf("%s\n", c)
			s += fmt.Sprintf("  ]]></content>\n")
			s += fmt.Sprintf("</entry>\n")
		}
	}
	s += fmt.Sprintf("</feed>\n")

	body := []byte(s)

	return body, nil
}
func buildPage(dest string, ent config.Post, configs []config.Post) error {
	var data PageData
	data.SiteConfig = cfg
	data.Post = ent
	data.Web = true
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

	body, err := ExecuteTemplateByName("pre", data)
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
		if ent.Type == "atom" {
			var feed Feed
			feed.SiteTitle = cfg.Title
			feed.SiteURL = cfg.URL
			feed.SiteID = cfg.URL + "/"
			feed.Author = cfg.Author

			body, err := CreateAtomFeed(feed, configs)
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
