package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"log"
	"os"
	"strings"
)

type csvTable = [][]string

func getHeaders(s goquery.Selection) []string {
	return s.Find("tr").First().Find("th").Map(func(i int, ss *goquery.Selection) string {
		return strings.TrimSpace(ss.Text())
	})
}

func getRows(s goquery.Selection) []goquery.Selection {
	var rows []goquery.Selection
	s.Find("tr").Next().Each(func(i int, ss *goquery.Selection) {
		rows = append(rows, *ss)
	})
	return rows
}

func getCellString(s goquery.Selection) string {
	var cell string
	cell = strings.TrimSpace(s.Text())
	if len(cell) > 0 {
		return cell
	}
	cell = s.Find("a").Text()
	if len(cell) > 0 {
		return cell
	}
	cell, _ = s.Find("a").Attr("title")
	if len(cell) > 0 {
		return cell
	}
	return cell
}

func getData(s goquery.Selection) csvTable {
	rows := getRows(s)
	var data csvTable
	for _, row := range rows {
		tds := row.Find("td, th").Map(func(i int, ss *goquery.Selection) string {
			return getCellString(*ss)
		})
		data = append(data, tds)
	}
	return data
}

func processTable(s goquery.Selection) csvTable {
	headers := getHeaders(s)
	data := getData(s)
	return append(csvTable{headers}, data...)
}

func main() {
	var n = flag.Int("n", -1, "Index of table to print. Starts at 0. -1 prints all tables")
	var selector = flag.String("s", "table.wikitable", "The CSS selector to identify tables.")
	var o = flag.String("o", "", "If set will output to file(s). Will be in the form of <file>_<n>.csv if mulitple tables")
	flag.Parse()

	doc, err := goquery.NewDocumentFromReader(os.Stdin)
	if err != nil {
		log.Fatal("Error while reading html input")
		os.Exit(1)
	}
	tables := doc.Find(*selector)

	var csvTables []csvTable
	tables.Each(func(i int, s *goquery.Selection) {
		csvTable := processTable(*s)
		csvTables = append(csvTables, csvTable)
	})

	if *n > -1 {
		slice := csvTables[*n]
		csvTables = []csvTable{slice}
	}

	if *o != "" {
		if len(csvTables) == 1 {
			filename := fmt.Sprintf("%s.csv", *o)
			f, err := os.Create(filename)
			if err != nil {
				log.Fatal("Failed to open file for writing")
				os.Exit(1)
			}
			w := csv.NewWriter(f)
			w.WriteAll(csvTables[0])
		} else {
			for i, table := range csvTables {
				filename := fmt.Sprintf("%s_%d.csv", *o, i)
				f, err := os.Create(filename)
				if err != nil {
					log.Fatal("Failed to open file for writing")
					os.Exit(1)
				}
				w := csv.NewWriter(f)
				w.WriteAll(table)
			}
		}
	} else {
		w := csv.NewWriter(os.Stdout)

		for _, table := range csvTables {
			w.WriteAll(table)
		}
	}
}
