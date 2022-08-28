package gcurl

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	var tests = []struct {
		name     string
		given    string
		expected *Request
	}{
		{
			"simple get",
			"curl https://api.sloths.com",
			&Request{
				Method: http.MethodGet,
				URL:    "https://api.sloths.com",
				Header: map[string]string{},
			},
		},
		{
			"simple put",
			"curl -XPUT https://api.sloths.com/sloth/4",
			&Request{
				Method: http.MethodPut,
				URL:    "https://api.sloths.com/sloth/4",
				Header: map[string]string{},
			},
		},
		{
			"encoding gzip",
			`curl -H "Accept-Encoding: gzip" --compressed http://api.sloths.com`,
			&Request{
				Method: http.MethodGet,
				URL:    "http://api.sloths.com",
				Header: map[string]string{
					"Accept-Encoding": "gzip",
				},
			},
		},
		{
			"delete sloth",
			"curl -X DELETE https://api.sloths.com/sloth/4",
			&Request{
				Method: http.MethodDelete,
				URL:    "https://api.sloths.com/sloth/4",
				Header: map[string]string{},
			},
		},
		{
			"url encoded data",
			`curl -d "foo=bar" https://api.sloths.com/sloth/4`,
			&Request{
				Method: http.MethodPost,
				URL:    "https://api.sloths.com/sloth/4",
				Header: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
				Body:   "foo=bar",
			},
		},
		{
			"user agent",
			`curl -H "Accept: text/plain" --header "User-Agent: slothy" https://api.sloths.com`,
			&Request{
				Method: http.MethodGet,
				URL:    "https://api.sloths.com",
				Header: map[string]string{
					"Accept":     "text/plain",
					"User-Agent": "slothy",
				},
			},
		},
		{
			"cookie",
			`curl --cookie 'species=sloth;type=galactic' slothy https://api.sloths.com`,
			&Request{
				Method: http.MethodGet,
				URL:    "https://api.sloths.com",
				Header: map[string]string{
					"Cookie": "species=sloth;type=galactic",
				},
			},
		},
		{
			"location",
			`curl --location --request GET 'https://api.sloths.com/users?token=admin'`,
			&Request{
				Method: http.MethodGet,
				URL:    "https://api.sloths.com/users?token=admin",
				Header: map[string]string{},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			actual, err := Parse(tt.given)
			require.NoError(t, err)
			require.Equal(t, tt.expected, actual)
		})
	}
}
