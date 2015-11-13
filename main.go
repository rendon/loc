package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/goutil/ds"
)

type City struct {
	Name    string
	Country string
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("USAGE: %s <locations_file>", os.Args[0])
	}
	locFile := os.Args[1]

	buf, err := ioutil.ReadFile("data/country_codes.txt")
	if err != nil {
		log.Fatal(err)
	}
	re := regexp.MustCompile("\\s*,\\s*")
	countryCodes := make(map[string]string)
	lines := strings.Split(string(buf), "\n")
	for _, line := range lines {
		tokens := strings.Split(line, " ")
		if len(tokens) != 2 {
			continue
		}
		countryCodes[tokens[0]] = tokens[1]
		//log.Printf("%s -> %s\n", tokens[0], tokens[1])
	}

	countries := make(map[string]bool)
	trie := ds.NewTrie()
	buf, err = ioutil.ReadFile("data/GeoLite2-City-Locations-en.csv")
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
	buf, err = ioutil.ReadFile(locFile)
	if err != nil {
		log.Fatal(err)
	}
	locations := strings.Split(string(buf), "\n")
	for _, loc := range locations {
		loc := strings.ToLower(loc)
		tokens := re.Split(loc, -1)
		if len(tokens) == 2 {
			country := strings.Trim(tokens[1], " .,")
			// Case 1: city, [country]
			if countries[country] {
				fmt.Printf("1> %s: %s\n", loc, country)
				continue
			}
		}
		// Case 2: Exact match
		if v := trie.Find(loc); v != nil {
			fmt.Printf("2> %s: %v\n", loc, v)
			continue
		}

		// Case 3: [city], country
		if len(tokens) > 0 {
			if v := trie.Find(tokens[0]); v != nil {
				fmt.Printf("3> %s: %v\n", loc, v)
				continue
			}
		}

		// Case 4: by country code
		if len(tokens) == 2 {
			if countryCodes[tokens[1]] != "" {
				fmt.Printf("4> %s: %v\n", loc, countryCodes[tokens[1]])
			}
		}

		// Case 5: by city code

		// Case 6: approximate string matching
	}
}
