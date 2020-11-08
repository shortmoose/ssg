package util_test

import (
	"strings"
	"testing"

	"github.com/andreyvit/diff"
	"github.com/shortmoose/ssg/internal/config"
	"github.com/shortmoose/ssg/internal/util"
)

func TestCreateAtomFeedEmpty(t *testing.T) {
	var feed util.Feed
	feed.SiteURL = "http://test.test"
	feed.SiteTitle = "Test Site"
	feed.SiteID = "http://testid.test"
	feed.Author = "me"

	var posts []config.Post
	_, err := util.CreateAtomFeed(feed, posts)
	if err != nil {
		if !strings.Contains(err.Error(), "Can't create XML feed, no entries") {
			t.Errorf("Unexpected error creating atom feed: %v", err)
		}
		return
	}

	t.Errorf("Expected an error creating atom feed")
}

func TestCreateAtomFeed(t *testing.T) {
	var feed util.Feed
	feed.SiteURL = "http://test.test"
	feed.SiteTitle = "Test Site"
	feed.SiteID = "http://testid.test"
	feed.Author = "me"

	var entry config.Post
	entry.Date = "2020-07-29T13:40:21Z"
	entry.Title = "Post Title"
	entry.SitePath = "/postpath"
	entry.Content = []byte("Post Content")

	posts := []config.Post{entry}

	atom, err := util.CreateAtomFeed(feed, posts[0:1])
	if err != nil {
		t.Errorf("Error creating atom feed: %v", err)
		return
	}

	expected := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test Site</title>
  <link href="http://test.test/" />
  <updated>2020-07-29T13:40:21Z</updated>
  <id>http://testid.test</id>
<entry>
  <title>Post Title</title>
  <link href="http://test.test/postpath" />
  <updated>2020-07-29T13:40:21Z</updated>
  <id>http://test.test/postpath</id>
  <author><name>me</name></author>
  <content type="html"><![CDATA[
Post Content
  ]]></content>
</entry>
</feed>
`
	if expected != string(atom) {
		t.Errorf("Feed didn't match expected: %s", atom)
		t.Errorf("Expected: %s", diff.LineDiff(string(atom), expected))
	}
}
