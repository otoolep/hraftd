# Go Enterprise [![GoDoc](https://godoc.org/github.com/pascaldekloe/goe?status.svg)](https://godoc.org/github.com/pascaldekloe/goe) [![Build Status](https://travis-ci.org/pascaldekloe/goe.svg?branch=master)](https://travis-ci.org/pascaldekloe/goe)

Common enterprise features for the Go programming language (golang).

This is free and unencumbered software released into the
[public domain](http://creativecommons.org/publicdomain/zero/1.0).


## Expression Language [API](http://godoc.org/github.com/pascaldekloe/goe/el)

GoEL expressions provide error free access to Go types.
It serves as a lightweigth alternative to [unified EL](https://docs.oracle.com/javaee/5/tutorial/doc/bnahq.html), [SpEL](http://docs.spring.io/spring/docs/current/spring-framework-reference/html/expressions.html) or even [XPath](http://www.w3.org/TR/xpath), [CSS selectors](http://www.w3.org/TR/css3-selectors) and friends.

``` Go
func FancyOneLiners() {
	// Single field selection:
	upper, applicable := el.Bool(`/CharSet[0x1F]/isUpperCase`, x)

	// Escape path separator slash:
	warnings := el.Strings(`/Report/Stats["I\x2fO"]/warn[*]`, x)

	// Data modification:
	el.Assign(x, `/Nodes[7]/Cache/TTL`, 3600)
```

#### Performance

The implementation is optimized for performance. No need to precompile expressions.

```
# go test -bench=. -benchmem
PASS
BenchmarkLookups-8	 2000000	       717 ns/op	     194 B/op	       6 allocs/op
BenchmarkAssigns-8	 2000000	       997 ns/op	     277 B/op	       8 allocs/op
ok  	github.com/pascaldekloe/goe/el	7.622s
```


## Metrics [API](https://godoc.org/github.com/pascaldekloe/goe/metrics)

Yet another StatsD implementation.

``` Go
var Metrics = metrics.NewDummy()

func GetSomething(w http.ResponseWriter, r *http.Request) {
	Metrics.Seen("http.something.gets", 1)
	defer Metrics.Took("http.something.get", time.Now())
```


## Verification [API](http://godoc.org/github.com/pascaldekloe/goe/verify)

Test assertions on big objects can be cumbersome with ```reflect.DeepEqual``` and ```"Got %#v, want %#v"```.
Package `verify` offers convenience with reporting. For example `verify.Values(t, "character", got, want)` might print:

```
--- FAIL: TestValuesDemo (0.00s)
	demo_test.go:72: verification for character:
		/Novel[6]/Title: Got "Gold Finger", want "Goldfinger"
		                          ^
		/Film[20]/Year: Got 1953 (0x7a1), want 2006 (0x7d6)
```
