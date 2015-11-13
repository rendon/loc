package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"regexp"
	"strings"

	"github.com/goutil/ds"
)

type City struct {
	Name    string
	Country string
}

func main() {
	countries := make(map[string]bool)
	trie := ds.NewTrie()
	buf, err := ioutil.ReadFile("GeoLite2Cities/GeoLite2-City-Locations-en.csv")
	if err != nil {
		log.Fatal(err)
	}
	r := csv.NewReader(bytes.NewReader(buf))
	first := true
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if first {
			first = false
			continue
		}
		city := strings.ToLower(record[10])
		country := strings.ToLower(record[5])
		if city == "" || country == "" {
			continue
		}
		countries[country] = true
		if trie.Insert(city, City{city, country}) {
			//fmt.Printf("New city added: %s\n", c.Name)
		}
	}
	//tour(root, "")
	buf, err = ioutil.ReadFile("GeoLite2Cities/locations.txt")
	if err != nil {
		log.Fatal(err)
	}
	var locations = strings.Split(string(buf), "\n")
	re := regexp.MustCompile("\\s*,\\s*")
	for _, loc := range locations {
		tokens := re.Split(loc, -1)
		if len(tokens) == 2 {
			country := strings.Trim(tokens[1], " .,")
			if countries[country] {
				fmt.Printf("1> %s: %s\n", loc, country)
				continue
			}
			if v := trie.Find(tokens[0]); v != nil {
				fmt.Printf("2> %s: %v\n", loc, v)
				continue
			}
		}

		loc := strings.ToLower(loc)
		if v := trie.Find(loc); v != nil {
			fmt.Printf("3> %s: %v\n", loc, v)
			continue
		}
	}
}
