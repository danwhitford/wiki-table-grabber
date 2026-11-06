package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type csvTable = [][]string

func getHeaders(s *goquery.Selection) []string {
	return s.Find("tr").First().Find("th").Map(func(_ int, ss *goquery.Selection) string {
		return strings.TrimSpace(ss.Text())
	})
}

func getRows(s *goquery.Selection) []*goquery.Selection {
	var rows []*goquery.Selection
	s.Find("tr").Next().Each(func(_ int, ss *goquery.Selection) {
		rows = append(rows, ss)
	})
	return rows
}

func getCellString(s *goquery.Selection) string {
	if cell := strings.TrimSpace(s.Text()); cell != "" {
		return cell
	}
	if cell := s.Find("a").Text(); cell != "" {
		return cell
	}
	cell, _ := s.Find("a").Attr("title")
	return cell
}

func getData(s *goquery.Selection) csvTable {
	var data csvTable
	for _, row := range getRows(s) {
		data = append(data, row.Find("td, th").Map(func(_ int, ss *goquery.Selection) string {
			return getCellString(ss)
		}))
	}
	return data
}

func processTable(s *goquery.Selection) csvTable {
	return append(csvTable{getHeaders(s)}, getData(s)...)
}

// parseTables reads an HTML document from r and converts every table matching
// selector into a csvTable.
func parseTables(r io.Reader, selector string) ([]csvTable, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("reading html input: %w", err)
	}
	var tables []csvTable
	doc.Find(selector).Each(func(_ int, s *goquery.Selection) {
		tables = append(tables, processTable(s))
	})
	return tables, nil
}

// selectTable returns only the table at index n, or all tables if n is negative.
func selectTable(tables []csvTable, n int) ([]csvTable, error) {
	if n < 0 {
		return tables, nil
	}
	if n >= len(tables) {
		return nil, fmt.Errorf("table index %d out of range: found %d table(s)", n, len(tables))
	}
	return tables[n : n+1], nil
}

// outputFilenames returns the file name to use for each table given the
// output prefix.
func outputFilenames(prefix string, count int) []string {
	if count == 1 {
		return []string{prefix + ".csv"}
	}
	names := make([]string, count)
	for i := range names {
		names[i] = fmt.Sprintf("%s_%d.csv", prefix, i)
	}
	return names
}

func writeTables(w io.Writer, tables ...csvTable) error {
	cw := csv.NewWriter(w)
	for _, table := range tables {
		if err := cw.WriteAll(table); err != nil {
			return err
		}
	}
	return nil
}

func writeTableFile(name string, table csvTable) (err error) {
	f, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("opening %s for writing: %w", name, err)
	}
	defer func() {
		if cerr := f.Close(); err == nil && cerr != nil {
			err = fmt.Errorf("closing %s: %w", name, cerr)
		}
	}()
	if err := writeTables(f, table); err != nil {
		return fmt.Errorf("writing %s: %w", name, err)
	}
	return nil
}

func run(r io.Reader, w io.Writer, selector string, n int, outPrefix string) error {
	tables, err := parseTables(r, selector)
	if err != nil {
		return err
	}
	tables, err = selectTable(tables, n)
	if err != nil {
		return err
	}
	if outPrefix == "" {
		return writeTables(w, tables...)
	}
	for i, name := range outputFilenames(outPrefix, len(tables)) {
		if err := writeTableFile(name, tables[i]); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	n := flag.Int("n", -1, "Index of table to print. Starts at 0. -1 prints all tables")
	selector := flag.String("s", "table.wikitable", "The CSS selector to identify tables.")
	o := flag.String("o", "", "If set will output to file(s). Will be in the form of <file>_<n>.csv if mulitple tables")
	flag.Parse()

	if err := run(os.Stdin, os.Stdout, *selector, *n, *o); err != nil {
		log.Fatal(err)
	}
}
