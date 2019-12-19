package rest_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	"github.com/pascaldekloe/goe/rest"
)

func ExampleCRUDRepo_SetCreateFunc() {
	type Data struct {
		Version int64  `json:"version"`
		Msg     string `json:"msg"`
	}
	var v atomic.Value
	v.Store(&Data{})

	repo := rest.NewCRUD("/", "/Version")
	repo.SetCreateFunc(func(d *Data) (int64, error) {
		d.Version = time.Now().UnixNano()
		v.Store(d)
		return 42, nil
	})

	server := httptest.NewServer(repo)
	defer server.Close()

	res, err := http.Post(server.URL, "application/json", bytes.NewBufferString(`{"msg": "Hello World!"}`))
	if err != nil {
		panic(err)
	}
	fmt.Printf("Got HTTP %s %s: %s\n", res.Status, res.Header.Get("Location"), v.Load().(*Data).Msg)

	// Output: Got HTTP 201 Created /42: Hello World!
}

func ExampleCRUDRepo_SetReadFunc() {
	type Data struct {
		Version int64  `json:"version"`
		Msg     string `json:"msg"`
	}

	repo := rest.NewCRUD("/", "/Version")
	repo.SetReadFunc(func(id, version int64) (*Data, error) {
		if id != 42 {
			return nil, rest.ErrNotFound
		}
		return &Data{1456260879956532222, "Hello World!"}, nil
	})

	server := httptest.NewServer(repo)
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL+"/42", nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("If-None-Match", `"1456260879956532222"`)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Got HTTP %s: %s\n", res.Status, res.Header.Get("Content-Location"))

	// Output: Got HTTP 304 Not Modified: /42?v=1456260879956532222
}
