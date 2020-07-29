package post

import (
	"testing"
	// "bytes"
	// "fmt"
	// "io/ioutil"
	// "time"

	// "gopkg.in/yaml.v2"
)



// test := `<!-- CONFIG
// title: Title
// snippet: Snippet about the article
// date: 2020-03-09T22:00:00Z
// CONFIG -->

// <h2 id="subtitle-1">Subtitle 1</h2>
// <p>Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.</p>

// <h2 id="subtitle-2">Subtitle 2</h2>
// <p>Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.</p>
// <ul>
// <li>one</li>
// <li>Two</li>
// </ul>

// <p class="sig">Creator</p>
// <p class="date">Published on March 9, 2020</p>`


func TestStripPageConfig(t *testing.T) {
	testStr := []byte(`<!-- CONFIG
	frog
	CONFIG -->
	<p>Hello</p>`)

    output := stripPageConfig(testStr)
    if string(output) != `<p>Hello</p>` {
        t.Errorf("stripPageConfig failed to properly strip config")
    }
}

func TestStripPageConfig2(t *testing.T) {
	testStr := []byte(`<p>Hello</p>`)

    output := stripPageConfig(testStr)
    if string(output) != `<p>Hello</p>` {
        t.Errorf("stripPageConfig changed file with no config")
    }
}
