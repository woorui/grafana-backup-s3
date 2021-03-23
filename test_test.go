package main

import (
	"testing"
)

var mockdata = []struct {
	a      string
	b      string
	except string
}{
	{a: "/Users/wurui/grafana-backup/1616398799", b: "/Users/wurui/grafana-backup/1616398799/080", except: "/080"},
	{a: "/Users/wurui/grafana-backup/1616398799", b: "/Users/wurui/grafana-backup/1616398799", except: ""},
	{a: "", b: "", except: ""},
	{a: "/Users/wurui/grafana-backup/1616398799", b: "", except: "/Users/wurui/grafana-backup/1616398799"},
}

func Test_diffpath(t *testing.T) {
	for _, item := range mockdata {
		r := diffpath(item.a, item.b)
		if r != item.except {
			t.Errorf("a = %s, b = %s, except = %s, but got = %s", item.a, item.b, item.except, r)
		}
	}
}
