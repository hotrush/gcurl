package gcurl

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/mattn/go-shellwords"
)

var (
	ErrNotValidCurlCommand = errors.New("not a valid cURL command")
)

type Header map[string]string

type Request struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Header Header `json:"header"`
	Body   string `json:"body"`
}

func Parse(curl string) (*Request, error) {
	if strings.Index(curl, "curl ") != 0 {
		return nil, fmt.Errorf("%q: %w", curl, ErrNotValidCurlCommand)
	}

	// https://github.com/mattn/go-shellwords
	// https://github.com/tj/parse-curl.js
	args, err := shellwords.Parse(curl)
	if err != nil {
		return nil, err
	}

	args = rewrite(args)
	req := &Request{
		Method: http.MethodGet,
		Header: Header{},
	}

	var state string
	for _, arg := range args {
		switch true {
		case validURL(arg):
			req.URL = arg
			break

		case arg == "-A" || arg == "--user-agent":
			state = "user-agent"
			break

		case arg == "-H" || arg == "--header":
			state = "header"
			break

		case arg == "-d" || arg == "--data" || arg == "--data-ascii" || arg == "--data-raw":
			state = "data"
			break

		case arg == "-u" || arg == "--user":
			state = "user"
			break

		case arg == "-I" || arg == "--head":
			req.Method = "HEAD"
			break

		case arg == "-X" || arg == "--request":
			state = "method"
			break

		case arg == "-b" || arg == "--cookie":
			state = "cookie"
			break

		case len(arg) > 0:
			switch state {
			case "header":
				fields := parseField(arg)
				req.Header[fields[0]] = strings.TrimSpace(fields[1])
				state = ""
				break

			case "user-agent":
				req.Header["User-Agent"] = arg
				state = ""
				break

			case "data":
				if req.Method == http.MethodGet || req.Method == http.MethodHead {
					req.Method = "POST"
				}

				if !hasContentType(*req) {
					req.Header["Content-Type"] = "application/x-www-form-urlencoded"
				}

				if len(req.Body) == 0 {
					req.Body = arg
				} else {
					req.Body = req.Body + "&" + arg
				}

				state = ""
				break

			case "user":
				req.Header["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(arg))
				state = ""
				break

			case "method":
				req.Method = arg
				state = ""
				break

			case "cookie":
				req.Header["Cookie"] = arg
				state = ""
				break

			default:
				break
			}
		}

	}

	// format json body
	if val := req.Header["Content-Type"]; val == "application/json" {
		jsonData := make(map[string]interface{})
		if err := json.NewDecoder(strings.NewReader(req.Body)).Decode(&jsonData); err != nil {
			return nil, err

		}
		var buffer *bytes.Buffer
		encoder := json.NewEncoder(buffer)
		encoder.SetEscapeHTML(false)
		if err = encoder.Encode(jsonData); err != nil {
			return nil, err
		}
		req.Body = strings.ReplaceAll(buffer.String(), "\n", "")
	}
	return req, err
}

func rewrite(args []string) []string {
	res := make([]string, 0)
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "\n" {
			continue
		}

		if strings.Contains(arg, "\n") {
			arg = strings.ReplaceAll(arg, "\n", "")
		}

		// split request method
		if strings.Index(arg, "-X") == 0 {
			res = append(res, arg[0:2])
			res = append(res, arg[2:])
		} else {
			res = append(res, arg)
		}
	}
	return res
}

func validURL(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
	// _, err := url.Parse(u)
	// if err != nil {
	// 	return false
	// }
	// return true
}

func parseField(arg string) []string {
	index := strings.Index(arg, ":")
	return []string{arg[0:index], arg[index+2:]}
}

func hasContentType(req Request) bool {
	_, ok := req.Header["Content-Type"]
	return ok
}
