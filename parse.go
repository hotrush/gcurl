package gcurl

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/mattn/go-shellwords"
)

var ErrNotValidCurlCommand = errors.New("not a valid cURL command")

const (
	// Header map keys
	KeyContentType   = "content-type"
	KeyUserAgent     = "user-agent"
	KeyCookie        = "cookie"
	KeyAuthorization = "authorization"

	// Content-Types
	ContentTypeJSON = "application/json"
)

type Header map[string]string

type Request struct {
	Method  string `json:"method"`
	URL     string `json:"url"`
	Header  Header `json:"header"`
	Body    string `json:"body"`
	SkipTLS bool   `json:"skip_tls"`
	Timeout string `json:"timeout"`
}

func Parse(curl string) (*Request, error) {
	if strings.Index(curl, "curl ") != 0 {
		return nil, fmt.Errorf("%q: %w", curl, ErrNotValidCurlCommand)
	}

	args, err := shellwords.Parse(curl)
	if err != nil {
		return nil, err
	}

	args = sanitize(args)
	req := &Request{
		Method: http.MethodGet,
		Header: Header{},
	}

	var argType string
	for _, arg := range args {
		switch {
		case isURL(arg):
			req.URL = arg
			break
		case arg == "-A" || arg == "--user-agent":
			argType = "user-agent"
			break
		case arg == "-H" || arg == "--header":
			argType = "header"
			break
		case arg == "-d" || arg == "--data" || arg == "--data-ascii" || arg == "--data-raw":
			argType = "data"
			break
		case arg == "-u" || arg == "--user":
			argType = "user"
			break
		case arg == "-I" || arg == "--head":
			req.Method = "HEAD"
			break
		case arg == "-X" || arg == "--request":
			argType = "method"
			break
		case arg == "-b" || arg == "--cookie":
			argType = "cookie"
			break
		case arg == "-k" || arg == "--insecure":
			req.SkipTLS = true
			break
		case arg == "-m" || arg == "--max-time":
			argType = "timeout"
			break
		default:
			switch argType {
			case "header":
				key, val, _ := strings.Cut(arg, ":")
				req.Header[strings.ToLower(key)] = strings.TrimSpace(val)
				argType = ""
				break
			case "user-agent":
				req.Header[KeyUserAgent] = arg
				argType = ""
				break
			case "data":
				if req.Method == http.MethodGet || req.Method == http.MethodHead {
					req.Method = http.MethodPost
				}

				if _, ok := req.Header[KeyContentType]; !ok {
					req.Header[KeyContentType] = "application/x-www-form-urlencoded"
				}

				if len(req.Body) == 0 {
					req.Body = arg
				} else {
					req.Body = req.Body + "&" + arg
				}

				argType = ""
				break
			case "user":
				req.Header[KeyAuthorization] = "Basic " + base64.StdEncoding.EncodeToString([]byte(arg))
				argType = ""
				break
			case "method":
				req.Method = arg
				argType = ""
				break
			case "cookie":
				req.Header[KeyCookie] = arg
				argType = ""
				break
			case "timeout":
				req.Timeout = arg
				argType = ""
				break
			}
		}
	}

	// Format JSON body.
	if val := req.Header[KeyContentType]; val == ContentTypeJSON {
		data := make(map[string]interface{})
		if err := json.Unmarshal([]byte(req.Body), &data); err != nil {
			return nil, err
		}

		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		if err = enc.Encode(data); err != nil {
			return nil, err
		}
		req.Body = strings.ReplaceAll(buf.String(), "\n", "")
	}
	return req, err
}

func sanitize(args []string) []string {
	res := make([]string, 0)
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "\n" {
			continue
		}

		// Remove new lines characters.
		if strings.Contains(arg, "\n") {
			arg = strings.ReplaceAll(arg, "\n", "")
		}

		// Split method when -XMETHOD are concatenated.
		if strings.HasPrefix(arg, "-X") && len(arg) > 2 {
			res = append(res, arg[0:2])
			res = append(res, arg[2:])
			continue
		}
		res = append(res, arg)
	}
	return res
}

func isURL(u string) bool {
	matched, err := regexp.MatchString("^https?://.*$", u)
	return matched && err == nil
}
