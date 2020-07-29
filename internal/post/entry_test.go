package post

import (
	"testing"
	"reflect"
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
	type test struct {
        input string
        want  string
    }

    tests := []test{
        { 	input: `<!-- CONFIG
frog
CONFIG --> 
<p>Hello</p>`, 
		  	want: `<p>Hello</p>`,
		},
		{ 	input: `<p>Hello</p>`, 
			want: `<p>Hello</p>`,
		},
		{ 	input: `<!-- comment --> 
		<p>Hello</p>`, 
			want: `<!-- comment --> 
		<p>Hello</p>`,
		},
    }

    for _, tc := range tests {
        output := string(stripPageConfig([]byte(tc.input)))
        if !reflect.DeepEqual(tc.want, output) {
            t.Errorf("expected: %v, got: %v", tc.want, output)
        }
	}
}
