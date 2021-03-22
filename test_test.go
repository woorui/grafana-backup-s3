package main

import (
	"testing"
)

type diffpathTest struct {
	a      string
	b      string
	except string
}

func diffpathTestMockData() []diffpathTest {
	return []diffpathTest{
		{a: "/Users/wurui/grafana-backup/1616398799", b: "/Users/wurui/grafana-backup/1616398799/080", except: "/080"},
		{a: "/Users/wurui/grafana-backup/1616398799", b: "/Users/wurui/grafana-backup/1616398799", except: ""},
		{a: "", b: "", except: ""},
		{a: "/Users/wurui/grafana-backup/1616398799", b: "", except: "/Users/wurui/grafana-backup/1616398799"},
	}
}

func Test_diffpath(t *testing.T) {
	data := diffpathTestMockData()
	for _, item := range data {
		r := diffpath(item.a, item.b)
		if r != item.except {
			t.Errorf("a = %s, b = %s, except = %s, but got = %s", item.a, item.b, item.except, r)
		}
	}
}
