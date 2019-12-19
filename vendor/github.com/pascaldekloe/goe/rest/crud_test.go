package rest

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type Data struct {
	Version int64  `json:"version"`
	Msg     string `json:"msg"`
}

var goldenCases = []struct {
	Name           string
	Method         string
	Path           string
	Body           string
	Headers        map[string]string
	WantStatusCode int
	WantBody       string
	WantHeaders    map[string]string
	Operation      interface{}
}{

	{"create",
		"POST", "/", `{"msg": "hello"}`, map[string]string{"Content-Type": "application/json"},
		201, "", map[string]string{
			"Location": "/1001",
			"ETag":     `"1456260879956532222"`,
		},
		func(d *Data) (int64, error) {
			d.Version = 1456260879956532222
			return 1001, nil
		},
	},
	{"create fail & verify version untouched",
		"POST", "/", `{"version": 3, "msg": "hello"}`, map[string]string{"Content-Type": "application/json"},
		500, "error v3\n",
		map[string]string{
			"Location": "",
			"ETag":     "",
		},
		func(d *Data) (int64, error) {
			return 0, fmt.Errorf("error v%d", d.Version)
		},
	},

	// Verify call arguments:
	{"read latest fail",
		"GET", "/99", "", nil,
		500, "error 99 v0\n", nil,
		func(id, version int64) (*Data, error) {
			return nil, fmt.Errorf("error %d v%d", id, version)
		},
	},
	{"read version fail",
		"GET", "/99?v=2", "", nil,
		500, "error 99 v2\n", nil,
		func(id, version int64) (*Data, error) {
			return nil, fmt.Errorf("error %d v%d", id, version)
		},
	},

	// Verify return arguments:
	{"read not found",
		"GET", "/99?v=2", "", nil,
		404, "ID 99 not found\n", map[string]string{
			"Content-Location": "",
			"ETag":             "",
			"Last-Modified":    "",
		},
		func(id, version int64) (*Data, error) {
			return nil, ErrNotFound
		},
	},
	{"read version not found",
		"GET", "/99?v=2", "", nil,
		404, "version 2 not found (latest is 1)\n", map[string]string{
			"Content-Location": "",
			"ETag":             "",
			"Last-Modified":    "",
		},
		func(id, version int64) (*Data, error) {
			return &Data{Version: 1}, nil
		},
	},

	// Caching:
	{"read cache miss",
		"GET", "/99", "", map[string]string{"If-None-Match": `"1456249153812139289"`, "If-Modified-Since": "Tue, 23 Feb 2016 20:54:39 UTC"},
		200, "{\n\t\"version\": 1456260879956532222,\n\t\"msg\": \"hello 99 v0\"\n}\n", map[string]string{
			"Content-Location": "/99?v=1456260879956532222",
			"Allow":            "OPTIONS, GET, HEAD",
			"ETag":             `"1456260879956532222"`,
			"Last-Modified":    "Tue, 23 Feb 2016 20:54:39 UTC",
			"Content-Type":     "application/json;charset=UTF-8",
		},
		func(id, version int64) (*Data, error) {
			return &Data{1456260879956532222, fmt.Sprintf("hello %d v%d", id, version)}, nil
		},
	},
	{"read ETag cache hit",
		"GET", "/99", "", map[string]string{"If-None-Match": `"1456260879956532222"`},
		304, "", map[string]string{
			"Content-Location": "/99?v=1456260879956532222",
			"Allow":            "OPTIONS, GET, HEAD",
			"ETag":             `"1456260879956532222"`,
			"Last-Modified":    "", // forbidden by status code
		},
		func(id, version int64) (*Data, error) {
			return &Data{Version: 1456260879956532222}, nil
		},
	},
	{"read timestamp cache hit",
		"GET", "/99", "", map[string]string{"If-Modified-Since": "Tue, 23 Feb 2016 20:54:40 UTC"},
		304, "", map[string]string{
			"Content-Location": "/99?v=1456260879956532222",
			"Allow":            "OPTIONS, GET, HEAD",
			"ETag":             `"1456260879956532222"`,
			"Last-Modified":    "", // forbidden by status code
		},
		func(id, version int64) (*Data, error) {
			return &Data{Version: 1456260879956532222}, nil
		},
	},

	{"update fail",
		"PUT", "/99", `{"version": 2, "msg": "hello"}`, map[string]string{"Content-Type": "application/json"},
		500, "hello 99 v2\n", nil,
		func(id int64, d *Data) error {
			return fmt.Errorf("%s %d v%d", d.Msg, id, d.Version)
		},
	},
	{"update version match",
		"PUT", "/99", `{"version": 2, "msg": "hello"}`, map[string]string{"Content-Type": "application/json", "If-Match": `"1456260879956532222"`},
		200, "{\n\t\"version\": 1456264479956532222,\n\t\"msg\": \"hello\"\n}\n", map[string]string{
			"Content-Location": "/99?v=1456264479956532222",
			"Allow":            "OPTIONS, PUT",
			"Content-Type":     "application/json;charset=UTF-8",
			"ETag":             `"1456264479956532222"`,
			"Last-Modified":    "Tue, 23 Feb 2016 21:54:39 UTC",
		},
		func(id int64, d *Data) error {
			d.Version += int64(time.Hour)
			return nil
		},
	},

	{"delete fail",
		"DELETE", "/666", "", nil,
		500, "operation rejected\n", nil,
		func(id, version int64) error {
			if id != 666 || version != 0 {
				return fmt.Errorf("got ID %d v%d", id, version)
			}
			return fmt.Errorf("operation rejected")
		},
	},
	{"delete version match not found",
		"DELETE", "/666", "", map[string]string{"If-Match": `"1456260879956532222"`},
		404, "", nil,
		func(id, version int64) error {
			if id != 666 || version != 1456260879956532222 {
				return fmt.Errorf("got ID %d v%d", id, version)
			}
			return ErrNotFound
		},
	},
	{"delete version match optimistic lock lost",
		"DELETE", "/666", "", map[string]string{"If-Match": `"1456260879956532222"`},
		412, "lost optimistic lock\n", nil,
		func(id, version int64) error {
			if id != 666 || version != 1456260879956532222 {
				return fmt.Errorf("got ID %d v%d", id, version)
			}
			return ErrOptimisticLock
		},
	},
	{"delete version query optimistic lock lost",
		"DELETE", "/666?v=1456260879956532222", "", nil,
		405, "not the latest version\n", nil,
		func(id, version int64) error {
			if id != 666 || version != 1456260879956532222 {
				return fmt.Errorf("got ID %d v%d", id, version)
			}
			return ErrOptimisticLock
		},
	},
}

func TestGolden(t *testing.T) {
	for _, gold := range goldenCases {
		repo := NewCRUD("/", "/Version")
		switch gold.Method {
		case "POST":
			repo.SetCreateFunc(gold.Operation)
		case "GET":
			repo.SetReadFunc(gold.Operation)
		case "PUT":
			repo.SetUpdateFunc(gold.Operation)
		case "DELETE":
			repo.SetDeleteFunc(gold.Operation)
		default:
			t.Fatalf("%s: unknown HTTP method %q", gold.Name, gold.Method)
		}
		server := httptest.NewServer(repo)
		defer server.Close()

		var reqBody io.Reader
		if gold.Body != "" {
			reqBody = strings.NewReader(gold.Body)
		}
		req, err := http.NewRequest(gold.Method, server.URL+gold.Path, reqBody)
		if err != nil {
			t.Fatalf("%s: malformed request: %s", gold.Name, err)
		}
		for name, value := range gold.Headers {
			req.Header.Set(name, value)
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: HTTP exchange: %s", gold.Name, err)
		}

		if res.StatusCode != gold.WantStatusCode {
			buf := new(bytes.Buffer)
			if err := res.Write(buf); err != nil {
				t.Fatalf("%s: Read response: %s", gold.Name, err)
			}
			t.Logf("%s: response: %s", gold.Name, buf.String())

			t.Errorf("%s: Got HTTP %q, want HTTP %d", gold.Name, res.Status, gold.WantStatusCode)
		}

		for name, want := range gold.WantHeaders {
			if got := res.Header.Get(name); got != want {
				t.Errorf("%s: Got header %s value %q, want %q", gold.Name, name, got, want)
			}
		}

		if want := gold.WantBody; want != "" {
			buf := new(bytes.Buffer)
			if _, err := buf.ReadFrom(res.Body); err != nil {
				t.Fatalf("%s: Read response body: %s", gold.Name, err)
			}

			if got := buf.String(); got != want {
				t.Errorf("%s: Got body %q, want %q", gold.Name, got, want)
			}
		}
	}
}
