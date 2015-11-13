package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"
)

type City struct {
	Name    string
	Country string
}

type Node struct {
	Char  byte
	Next  map[byte]*Node
	Value City
}

var (
	root *Node
)

func insert(node *Node, city, country string, p int) bool {
	if node == nil {
		return false
	}
	if p == len(city) {
		node.Value = City{city, country}
		return false
	}
	newWord := false
	char := city[p]
	if node.Next[char] == nil {
		node.Next[char] = &Node{
			Char: char,
			Next: make(map[byte]*Node),
		}
		newWord = true
	}
	newWord = insert(node.Next[char], city, country, p+1) || newWord
	return newWord
}

func addCity(city, country string) bool {
	return insert(root, city, country, 0)
}

func tour(node *Node, word string) {
	count := 0
	for k, v := range node.Next {
		count += 1
		w := word + string(k)
		tour(v, w)
	}
	if count == 0 {
		fmt.Printf("> %v: %v\n", word, node.Value)
	}
}

func find(node *Node, city string, p int) *City {
	if node == nil {
		return nil
	}
	if p == len(city) {
		if node.Value.Name == city {
			return &node.Value
		}
		return nil
	}
	char := city[p]
	if node.Next[char] == nil {
		return nil
	}
	return find(node.Next[char], city, p+1)
}

func findCity(city string) *City {
	return find(root, city, 0)
}

func main() {
	root = &Node{Next: make(map[byte]*Node)}
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
		var c = City{
			Name:    strings.ToLower(record[10]),
			Country: strings.ToLower(record[5]),
		}
		if c.Name == "" || c.Country == "" {
			continue
		}
		if addCity(c.Name, c.Country) {
			//fmt.Printf("New city added: %s\n", c.Name)
		}
	}
	//tour(root, "")
	buf, err = ioutil.ReadFile("GeoLite2Cities/locations.txt")
	if err != nil {
		log.Fatal(err)
	}
	var locations = strings.Split(string(buf), "\n")
	for _, loc := range locations {
		loc := strings.ToLower(loc)
		v := findCity(loc)
		if v != nil {
			fmt.Printf("%s: %s\n", loc, *v)
		}
	}
}
