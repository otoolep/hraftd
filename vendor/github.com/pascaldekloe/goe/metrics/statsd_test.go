package metrics

import (
	"bytes"
	"io/ioutil"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestSeen(t *testing.T) {
	buf := new(bytes.Buffer)

	d := NewStatsD(buf, 0)
	d.Seen("gorets", 2)
	d.Seen("gorets", 40)
	d.Seen("gorets", 800)
	d.Seen("gorets", 1600)

	time.Sleep(30 * time.Millisecond)
	want := "gorets:2|cgorets:40|cgorets:800|cgorets:1600|c"
	if got := buf.String(); got != want {
		t.Errorf("Got %q, want %q", got, want)
	}
}

func TestTook(t *testing.T) {
	start := time.Now()
	buf := new(bytes.Buffer)

	d := NewStatsD(buf, 0)
	d.Took("nglork", start)
	d.Took("nglork", start)

	time.Sleep(30 * time.Millisecond)
	want := regexp.MustCompile(`^glork:[0-9]+|msglork:[0-9]+|ms$`)
	if got := buf.String(); !want.MatchString(got) {
		t.Errorf("Got %q, want match %q", got, want)
	}
}

func TestKeyPrefix(t *testing.T) {
	start := time.Now()
	buf := new(bytes.Buffer)

	d := NewStatsD(buf, 0)
	d.KeyPrefix("group.")
	d.Seen("count", 99)
	d.Took("time", start)

	time.Sleep(30 * time.Millisecond)
	want := regexp.MustCompile(`^group\.count:99|cgroup\.time:[0-9]+|ms$`)
	if got := buf.String(); !want.MatchString(got) {
		t.Errorf("Got %q, want match %q", got, want)
	}
}

func TestBatch(t *testing.T) {
	buf := new(bytes.Buffer)

	message := "counter:5|c"
	messagesInPacket := StatsDPackMax / (len(message) + 1)
	packet := message + strings.Repeat("\n"+message, messagesInPacket-1)

	d := NewStatsD(buf, 50*time.Millisecond)
	for i := 0; i < 2*messagesInPacket; i++ {
		d.Seen("counter", 5)
	}

	time.Sleep(100 * time.Millisecond)
	if got, want := buf.String(), strings.Repeat(packet, 2); got != want {
		t.Errorf("Got:\n%q\nWant:\n%q", got, want)
	}
}

func BenchmarkSeen(b *testing.B) {
	d := NewStatsD(ioutil.Discard, time.Millisecond)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d.Seen("bench.key", i)
	}
}

func BenchmarkTook(b *testing.B) {
	start := time.Now()
	d := NewStatsD(ioutil.Discard, time.Millisecond)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d.Took("bench.key", start)
	}
}
