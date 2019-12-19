package rest

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pascaldekloe/goe/el"
)

var (
	// ErrNotFound signals that the resource is absent.
	// See CRUDRepo's SetReadFunc, SetUpdateFunc and SetDeleteFunc for the details.
	ErrNotFound = errors.New("no such entry")

	// ErrOptimisticLock signals that the latest version does not match the request.
	// See CRUDRepo's SetUpdateFunc and SetDeleteFunc for the details.
	ErrOptimisticLock = errors.New("lost optimistic lock")
)

var (
	keyType   = reflect.TypeOf(int64(0))
	errorType = reflect.TypeOf((*error)(nil)).Elem()
)

// CRUDRepo is a REST repository.
type CRUDRepo struct {
	// mountLocation is the root path of this repository.
	mountLoc string

	// versionPath is the GoEL expression to the data's version int64.
	versionPath string

	create, read, update, delete *reflect.Value

	dataType reflect.Type
}

// NewCRUD returns a new REST repository for the CRUD operations.
// The mountLocation specifies the root for CRUDRepo.ServeHTTP.
// The versionPath is a GoEL expression to the date version (in the data type).
//
// It's operation is based on two assumptions.
// 1) Identifiers are int64.
// 2) Versions are int64 unix timestamps in nanoseconds.
func NewCRUD(mountLocation, versionPath string) *CRUDRepo {
	return &CRUDRepo{
		mountLoc:    path.Clean(mountLocation),
		versionPath: versionPath,
	}
}

// SetCreateFunc enables create support.
// The method panics on any of the following conditions.
// 1) f does not match signature func(data T) (id int64, err error)
// 2) Data type T does not match the other CRUD operations.
// 3) Data type T is not a pointer.
//
// It is the responsibility of f to set the version.
func (repo *CRUDRepo) SetCreateFunc(f interface{}) {
	v := reflect.ValueOf(f)
	repo.create = &v

	t := v.Type()
	if t.Kind() != reflect.Func || t.NumIn() != 1 || t.NumOut() != 2 || t.Out(0) != keyType || !t.Out(1).Implements(errorType) {
		log.Panic("create is not a func(data T, id int64) error")
	}
	repo.setDataType(t.In(0))
}

// SetReadFunc enables read support.
// The method panics on any of the following conditions.
// 1) f does not match signature func(id, version int64) (hit T, err error)
// 2) Data type T does not match the other CRUD operations.
// 3) Data type T is not a pointer.
//
// When the id is not found f must return ErrNotFound.
// The version must be honored and the latest version should be served as a fallback.
func (repo *CRUDRepo) SetReadFunc(f interface{}) {
	v := reflect.ValueOf(f)
	repo.read = &v

	t := v.Type()
	if t.Kind() != reflect.Func || t.NumIn() != 2 || t.In(0) != keyType || t.In(1) != keyType || t.NumOut() != 2 || !t.Out(1).Implements(errorType) {
		log.Panic("read is not a func(id, version int64) (T, error)")
	}
	repo.setDataType(t.Out(0))
}

// SetUpdateFunc enables update support.
// The method panics on any of the following conditions.
// 1) f does not match signature func(id int64, data T) (error)
// 2) Data type T does not match the other CRUD operations.
// 3) Data type T is not a pointer.
//
// When the id is not found f must return ErrNotFound.
// When the data's version is not equal to 0 and version does not match the latest
// one available then f must skip normal operation and return ErrOptimisticLock.
// It is the responsibility of f to set the new version.
func (repo *CRUDRepo) SetUpdateFunc(f interface{}) {
	v := reflect.ValueOf(f)
	repo.update = &v

	t := v.Type()
	if t.Kind() != reflect.Func || t.NumIn() != 2 || t.In(0) != keyType || t.NumOut() != 1 || !t.Out(0).Implements(errorType) {
		log.Panic("update is not a func(id int64, data T) error")
	}
	repo.setDataType(t.In(1))

}

// SetUpdateFunc enables update support.
// The method panics when f does not match signature func(id, version int64) error.
//
// When the id is not found f must return ErrNotFound.
// When the version is not equal to 0 and version does not match the latest
// one available then f must skip normal operation and return ErrOptimisticLock.
func (repo *CRUDRepo) SetDeleteFunc(f interface{}) {
	v := reflect.ValueOf(f)
	repo.delete = &v

	t := v.Type()
	if t.Kind() != reflect.Func || t.NumIn() != 2 || t.In(0) != keyType || t.In(1) != keyType && t.NumOut() != 1 || !t.Out(0).Implements(errorType) {
		log.Panic("delete is not a func(id, version int64) error")
	}
}

func (repo *CRUDRepo) setDataType(t reflect.Type) {
	if t.Kind() != reflect.Ptr {
		log.Panicf("goe rest: CRUD operation's data type %s must be a pointer", t)
	}

	switch repo.dataType {
	case nil:
		repo.dataType = t

		if n := el.Assign(reflect.New(t).Interface(), repo.versionPath, 99); n != 1 {
			log.Panicf("goe rest: version path %q matches %d element on type %s", repo.versionPath, n, t)
		}
	case t:
		// do nothing
	default:
		log.Panicf("goe rest: CRUD operation's data type %s does not match %s", t, repo.dataType)
	}
}

// ServeHTTP honors the http.Handler interface for the mount point provided with NewCRUD.
// For now only JSON is supported.
func (repo *CRUDRepo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := path.Clean(r.URL.Path)
	if !strings.HasPrefix(p, repo.mountLoc) {
		log.Printf("goe rest: basepath %q mismatch with %q from %q", repo.mountLoc, p, r.URL.String())
		http.Error(w, "path mismatch", http.StatusNotFound)
		return
	}

	if len(p) == len(repo.mountLoc) {
		switch r.Method {
		default:
			if repo.create != nil {
				w.Header().Set("Allow", "POST")
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else {
				http.Error(w, "", http.StatusNotFound)
			}
		case "POST":
			if repo.create != nil {
				repo.serveCreate(w, r)
			} else {
				http.Error(w, "", http.StatusNotFound)
			}
		}
		return
	}

	if i := len(repo.mountLoc); p[i] == '/' {
		p = p[i+1:]
	} else {
		p = p[i:]
	}
	if i := strings.IndexByte(p, '/'); i >= 0 {
		http.Error(w, fmt.Sprintf("goe rest: no such subdirectory: %q", p), http.StatusNotFound)
		return
	}

	id, err := strconv.ParseInt(p, 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("goe rest: malformed ID: %s", err), http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET", "HEAD":
		if repo.read != nil {
			repo.serveRead(w, r, id)
			return
		}
	case "PUT":
		if repo.update != nil {
			repo.serveUpdate(w, r, id)
			return
		}
	case "DELETE":
		if repo.delete != nil {
			repo.serveDelete(w, r, id)
			return
		}
	case "OPTIONS":
		w.Header().Set("Allow", repo.resourceMethods())
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Allow", repo.resourceMethods())
	w.WriteHeader(http.StatusMethodNotAllowed)
}

// resourceMethods lists the HTTP methods served on r's resources.
func (r *CRUDRepo) resourceMethods() string {
	a := make([]string, 1, 5)
	a[0] = "OPTIONS"
	if r.read != nil {
		a = append(a, "GET", "HEAD")
	}
	if r.update != nil {
		a = append(a, "PUT")
	}
	if r.delete != nil {
		a = append(a, "DELETE")
	}
	return strings.Join(a, ", ")
}

func (repo *CRUDRepo) serveCreate(w http.ResponseWriter, r *http.Request) {
	v := reflect.New(repo.dataType)
	if !ReceiveJSON(v.Interface(), r, w) {
		return
	}

	result := repo.create.Call([]reflect.Value{v.Elem()})
	if !result[1].IsNil() {
		err := result[1].Interface().(error)
		log.Print("goe/rest: create: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	loc := *r.URL // copy
	loc.Path = path.Join(loc.Path, strconv.FormatInt(result[0].Int(), 10))
	loc.RawQuery = ""
	loc.Fragment = ""

	h := w.Header()
	h.Set("Location", loc.String())

	version, _ := el.Int(repo.versionPath, v.Interface())
	h.Set("ETag", fmt.Sprintf(`"%d"`, version))

	w.WriteHeader(http.StatusCreated)
}

func (repo *CRUDRepo) serveRead(w http.ResponseWriter, r *http.Request, id int64) {
	versionReq, ok := versionQuery(r, w)
	if !ok {
		return
	}

	result := repo.read.Call([]reflect.Value{reflect.ValueOf(id), reflect.ValueOf(int64(versionReq))})
	if !result[1].IsNil() {
		switch err := result[1].Interface().(error); err {
		case ErrNotFound:
			http.Error(w, fmt.Sprintf("ID %d not found", id), http.StatusNotFound)
		default:
			log.Print("goe/rest: read: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	version, _ := el.Int(repo.versionPath, result[0].Interface())
	if versionReq != 0 && version != versionReq {
		http.Error(w, fmt.Sprintf("version %d not found (latest is %d)", versionReq, version), http.StatusNotFound)
		return
	}

	h := w.Header()

	etag := fmt.Sprintf(`"%d"`, version)
	h.Set("ETag", etag)

	loc := *r.URL // copy
	loc.RawQuery = fmt.Sprintf("v=%d", version)
	loc.Fragment = ""
	h.Set("Content-Location", loc.String())

	h.Set("Allow", repo.resourceMethods())

	// BUG(pascaldekloe): No support for multiple entity tags in If-None-Match header.
	for _, s := range r.Header["If-None-Match"] {
		if s == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	for _, s := range r.Header["If-Modified-Since"] {
		t, err := time.Parse(time.RFC1123, s)
		if err != nil {
			http.Error(w, fmt.Sprintf("If-Unmodified-Since header %q not RFC1123 compliant: %s", s, err), http.StatusBadRequest)
			return
		}
		// Round down to RFC 1123 resolution:
		resolution := int64(time.Second)
		if t.After(time.Unix(0, (version/resolution)*resolution)) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	timestamp := time.Unix(0, version)
	h.Set("Last-Modified", timestamp.In(time.UTC).Format(time.RFC1123))

	if r.Method != "HEAD" {
		ServeJSON(w, http.StatusOK, result[0].Interface())
	}
}

func (repo *CRUDRepo) serveUpdate(w http.ResponseWriter, r *http.Request, id int64) {
	v := reflect.New(repo.dataType)
	if !ReceiveJSON(v.Interface(), r, w) {
		return
	}

	queryVersion, ok := versionQuery(r, w)
	if !ok {
		return
	}

	matchVersion, ok := versionMatch(r, w)
	if !ok {
		return
	}

	var version int64
	switch {
	case queryVersion == 0:
		version = matchVersion
	case matchVersion == 0:
		version = queryVersion
	case queryVersion == matchVersion:
		version = matchVersion
	default:
		http.Error(w, fmt.Sprintf("query parameter v %d does not match If-Match header %d", queryVersion, matchVersion), http.StatusPreconditionFailed)
		return
	}
	if version != 0 {
		el.Assign(v.Interface(), repo.versionPath, version)
	}

	result := repo.update.Call([]reflect.Value{reflect.ValueOf(id), v.Elem()})
	if !result[0].IsNil() {
		switch err := result[0].Interface().(error); err {
		case ErrNotFound:
			http.Error(w, "", http.StatusNotFound)
		case ErrOptimisticLock:
			if matchVersion != 0 {
				http.Error(w, err.Error(), http.StatusPreconditionFailed)
				return
			}
			http.Error(w, "not the latest version", http.StatusMethodNotAllowed)
		default:
			log.Printf("goe rest: update %d v%d: %s", id, version, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	h := w.Header()

	version, _ = el.Int(repo.versionPath, v.Interface())
	h.Set("ETag", fmt.Sprintf(`"%d"`, version))
	h.Set("Last-Modified", time.Unix(0, version).In(time.UTC).Format(time.RFC1123))

	h.Set("Allow", repo.resourceMethods())

	loc := *r.URL // copy
	loc.RawQuery = fmt.Sprintf("v=%d", version)
	loc.Fragment = ""
	h.Set("Content-Location", loc.String())

	ServeJSON(w, http.StatusOK, v.Interface())
}

func (repo *CRUDRepo) serveDelete(w http.ResponseWriter, r *http.Request, id int64) {
	queryVersion, ok := versionQuery(r, w)
	if !ok {
		return
	}

	matchVersion, ok := versionMatch(r, w)
	if !ok {
		return
	}

	var version int64
	switch {
	case queryVersion == 0:
		version = matchVersion
	case matchVersion == 0:
		version = queryVersion
	case queryVersion != matchVersion:
		http.Error(w, fmt.Sprintf("query parameter v %d does not match If-Match header %d", queryVersion, matchVersion), http.StatusPreconditionFailed)
		return
	default:
		version = matchVersion
	}

	result := repo.delete.Call([]reflect.Value{reflect.ValueOf(id), reflect.ValueOf(version)})
	if !result[0].IsNil() {
		switch err := result[0].Interface().(error); err {
		case ErrNotFound:
			http.Error(w, "", http.StatusNotFound)
		case ErrOptimisticLock:
			if matchVersion != 0 {
				http.Error(w, err.Error(), http.StatusPreconditionFailed)
				return
			}
			http.Error(w, "not the latest version", http.StatusMethodNotAllowed)
		default:
			log.Printf("goe rest: delete %d v%d: %s", id, version, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)

}

// versionQuery parses URL parameter v or it returns ok false on error.
func versionQuery(r *http.Request, w http.ResponseWriter) (version int64, ok bool) {
	switch params := r.URL.Query()["v"]; len(params) {
	case 0:
		return 0, true
	case 1:
		i, err := strconv.ParseInt(params[0], 10, 64)
		if err != nil {
			http.Error(w, "query parameter v: malformed version number", http.StatusNotFound)
			return 0, false
		}
		return i, true
	default:
		http.Error(w, "multiple version query parameters", http.StatusBadRequest)
		return 0, false
	}
}

// versionMatch parses the If-Match header or it returns ok false on error.
func versionMatch(r *http.Request, w http.ResponseWriter) (version int64, ok bool) {
	// BUG(pascaldekloe): No support for multiple entity tags in If-Match header.

	tags := strings.Join(r.Header["If-Match"], ", ")
	if tags == "" || tags == "*" {
		return 0, true
	}

	const linearWhiteSpace = " \t"
	tag := strings.Trim(tags, linearWhiteSpace)
	if tag[0] != '"' || tag[len(tag)-1] != '"' {
		http.Error(w, fmt.Sprintf("need opaque tags in If-Match header %q", tag), http.StatusBadRequest)
		return 0, false
	}
	s := tag[1 : len(tag)-1]

	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("malformed or unknow tag in If-Match header %q", tag), http.StatusPreconditionFailed)
		return 0, false
	}
	return i, true
}
