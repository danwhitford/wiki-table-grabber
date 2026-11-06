package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

const sampleHTML = `<html><body>
<table class="wikitable">
  <tr><th> Name </th><th>Country</th><th>Flag</th></tr>
  <tr><th>Alice</th><td> UK </td><td><a href="/uk" title="United Kingdom"></a></td></tr>
  <tr><th>Bob</th><td><a href="/fr">France</a></td><td><a href="/fr" title="France"></a></td></tr>
</table>
<table class="other">
  <tr><th>Ignored</th></tr>
  <tr><td>row</td></tr>
</table>
<table class="wikitable">
  <tr><th>A</th><th>B</th></tr>
  <tr><td>1</td><td>2</td></tr>
</table>
</body></html>`

var (
	firstTable = csvTable{
		{"Name", "Country", "Flag"},
		{"Alice", "UK", "United Kingdom"},
		{"Bob", "France", "France"},
	}
	secondTable = csvTable{
		{"A", "B"},
		{"1", "2"},
	}
)

func mustSelection(t *testing.T, html string) *goquery.Selection {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	return doc.Selection
}

func TestGetCellString(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{"plain text is trimmed", `<table><tr><td>  hello  </td></tr></table>`, "hello"},
		{"link text", `<table><tr><td><a href="#">link</a></td></tr></table>`, "link"},
		{"falls back to link title", `<table><tr><td><a href="#" title="Title"></a></td></tr></table>`, "Title"},
		{"empty cell", `<table><tr><td></td></tr></table>`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cell := mustSelection(t, tt.html).Find("td")
			if got := getCellString(cell); got != tt.want {
				t.Errorf("getCellString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetHeaders(t *testing.T) {
	table := mustSelection(t, sampleHTML).Find("table.wikitable").First()
	want := []string{"Name", "Country", "Flag"}
	if got := getHeaders(table); !reflect.DeepEqual(got, want) {
		t.Errorf("getHeaders() = %q, want %q", got, want)
	}
}

func TestParseTables(t *testing.T) {
	got, err := parseTables(strings.NewReader(sampleHTML), "table.wikitable")
	if err != nil {
		t.Fatal(err)
	}
	want := []csvTable{firstTable, secondTable}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseTables() = %q, want %q", got, want)
	}
}

func TestParseTablesCustomSelector(t *testing.T) {
	got, err := parseTables(strings.NewReader(sampleHTML), "table.other")
	if err != nil {
		t.Fatal(err)
	}
	want := []csvTable{{{"Ignored"}, {"row"}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseTables() = %q, want %q", got, want)
	}
}

func TestParseTablesNoMatches(t *testing.T) {
	got, err := parseTables(strings.NewReader(`<p>no tables</p>`), "table.wikitable")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("parseTables() = %q, want no tables", got)
	}
}

func TestSelectTable(t *testing.T) {
	tables := []csvTable{firstTable, secondTable}

	if got, err := selectTable(tables, -1); err != nil || !reflect.DeepEqual(got, tables) {
		t.Errorf("selectTable(-1) = %q, %v; want all tables", got, err)
	}
	if got, err := selectTable(tables, 1); err != nil || !reflect.DeepEqual(got, []csvTable{secondTable}) {
		t.Errorf("selectTable(1) = %q, %v; want second table", got, err)
	}
	if _, err := selectTable(tables, 2); err == nil {
		t.Error("selectTable(2) expected out of range error, got nil")
	}
}

func TestOutputFilenames(t *testing.T) {
	if got, want := outputFilenames("out", 1), []string{"out.csv"}; !reflect.DeepEqual(got, want) {
		t.Errorf("outputFilenames(1) = %q, want %q", got, want)
	}
	if got, want := outputFilenames("out", 3), []string{"out_0.csv", "out_1.csv", "out_2.csv"}; !reflect.DeepEqual(got, want) {
		t.Errorf("outputFilenames(3) = %q, want %q", got, want)
	}
}

func TestRunStdout(t *testing.T) {
	var buf bytes.Buffer
	if err := run(strings.NewReader(sampleHTML), &buf, "table.wikitable", -1, ""); err != nil {
		t.Fatal(err)
	}
	want := "Name,Country,Flag\nAlice,UK,United Kingdom\nBob,France,France\nA,B\n1,2\n"
	if got := buf.String(); got != want {
		t.Errorf("run() output = %q, want %q", got, want)
	}
}

func TestRunSingleTableStdout(t *testing.T) {
	var buf bytes.Buffer
	if err := run(strings.NewReader(sampleHTML), &buf, "table.wikitable", 1, ""); err != nil {
		t.Fatal(err)
	}
	if got, want := buf.String(), "A,B\n1,2\n"; got != want {
		t.Errorf("run() output = %q, want %q", got, want)
	}
}

func TestRunIndexOutOfRange(t *testing.T) {
	if err := run(strings.NewReader(sampleHTML), &bytes.Buffer{}, "table.wikitable", 5, ""); err == nil {
		t.Error("run() expected error for out of range index, got nil")
	}
}

func TestRunWritesFiles(t *testing.T) {
	prefix := filepath.Join(t.TempDir(), "out")
	if err := run(strings.NewReader(sampleHTML), &bytes.Buffer{}, "table.wikitable", -1, prefix); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		prefix + "_0.csv": "Name,Country,Flag\nAlice,UK,United Kingdom\nBob,France,France\n",
		prefix + "_1.csv": "A,B\n1,2\n",
	} {
		got, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestRunWritesSingleFile(t *testing.T) {
	prefix := filepath.Join(t.TempDir(), "out")
	if err := run(strings.NewReader(sampleHTML), &bytes.Buffer{}, "table.wikitable", 1, prefix); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(prefix + ".csv")
	if err != nil {
		t.Fatal(err)
	}
	if want := "A,B\n1,2\n"; string(got) != want {
		t.Errorf("out.csv = %q, want %q", got, want)
	}
}
