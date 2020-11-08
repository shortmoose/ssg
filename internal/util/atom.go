package util

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"strings"
	"text/template"

	"github.com/shortmoose/ssg/internal/config"
)

// feed TODO
type Feed struct {
	SiteURL   string
	SiteTitle string
	SiteID    string
	Author    string
}

type PageData struct {
	config.Entry

	SiteConfig config.Site
	Pages      map[string]config.Entry
	Body       string
	Web        bool
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
	t, err := template.ParseGlob("templates/*")
	if err != nil {
		// TODO: There must be a better way to do this.
		if !strings.Contains(err.Error(), "matches no files") {
			return nil, err
		}
		t = template.New("zoo")
	}

	tmpl, err := t.New("x").Parse(templateText)

	out := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(out, "x", data)
	if err != nil {
		return nil, err
	}

	return out.Bytes(), err
}

func CreateAtomFeed(feed Feed, configs []config.Entry) ([]byte, error) {
	ents := []config.Entry{}
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
			data.Entry = e
			data.Pages = make(map[string]config.Entry)

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
