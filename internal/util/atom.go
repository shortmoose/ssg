package util

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/shortmoose/ssg/internal/post"
)

// feed TODO
type Feed struct {
	SiteURL   string
	SiteTitle string
	SiteID    string
	Author    string
	ImageURL  string
}

func CreateAtomFeed(feed Feed, configs []post.Entry) ([]byte, error) {
	ents := []post.Entry{}
	for i := range configs {
		if configs[i].Date != "" {
			ents = append(ents, configs[i])
		}
	}
	if len(ents) == 0 {
		return nil, fmt.Errorf("Can't create XML feed, no entries")
	}
	sort.Sort(post.ByDate(ents))

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
			s += fmt.Sprintf("%s\n", e.Content)
			s += fmt.Sprintf("  ]]></content>\n")
			s += fmt.Sprintf("</entry>\n")
		}
	}
	s += fmt.Sprintf("</feed>\n")

	body := []byte(s)
	body = bytes.ReplaceAll(body, []byte("/img/"), []byte(feed.ImageURL+"/"))
	body = bytes.ReplaceAll(body, []byte("/pdf/"), []byte(feed.ImageURL+"/"))
	body = bytes.ReplaceAll(body, []byte("href=\"/"), []byte("href=\""+feed.SiteURL+"/"))

	return body, nil
}
