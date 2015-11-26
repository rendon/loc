package main

import (
	"database/sql"
	logger "log"
	"os"
	"regexp"
	"strings"

	"github.com/goutil/ds"
	_ "github.com/mattn/go-sqlite3"
)

type Location struct {
	Continent        string
	Country          string
	ShortCountryCode string
	LongCountryCode  string
	City             string
}

var (
	splitRe   = regexp.MustCompile(`\\s*[,|\-/]\\s*`)
	punctRe   = regexp.MustCompile("[,.;:]")
	accentRe  = regexp.MustCompile("[áéíóú`]")
	specialRe = regexp.MustCompile(`[♥✈️]'"`)

	continentTrie = ds.NewTrie()
	countryTrie   = ds.NewTrie()
	cityTrie      = ds.NewTrie()

	countryCodes = make(map[string]string)
)

func init() {
	log = logger.New(os.Stderr, "", 0)
	db, err := sql.Open("sqlite3", "data/ccc.db")
	if err != nil {
		log.Fatal(err)
	}
	query := `
SELECT  continents.name AS continent, countries.name AS country, 
        countries.short_code AS short, countries.long_code AS long,
        cities.name AS city
FROM    continents, countries, cities
WHERE   continents.id = countries.continent_id AND
        cities.country_id = countries.id;
	`
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		var l Location
		err = rows.Scan(&l.Continent, &l.Country, &l.ShortCountryCode,
			&l.LongCountryCode, &l.City)
		if err != nil {
			log.Println(err)
			continue
		}
		l.Continent = strings.ToLower(l.Continent)
		l.Country = strings.ToLower(l.Country)
		l.City = strings.ToLower(l.City)

		continentTrie.Insert(l.Continent, &l)
		countryTrie.Insert(l.Country, &l)
		cityTrie.Insert(l.City, &l)
	}
	log.Printf("Finishing init.")
}

func normalizeLocation(loc string) *Location {
	loc = strings.ToLower(loc)
	loc = strings.Replace(loc, "à", "a", -1)
	loc = strings.Replace(loc, "è", "e", -1)
	loc = strings.Replace(loc, "ì", "i", -1)
	loc = strings.Replace(loc, "ò", "o", -1)
	loc = strings.Replace(loc, "ù", "u", -1)

	l, ok := continentTrie.Find(loc).(*Location)
	if ok && l != nil {
		return l
	}

	l, ok = countryTrie.Find(loc).(*Location)
	if ok && l != nil {
		return l
	}

	l, ok = countryTrie.Find(loc).(*Location)
	if ok && l != nil {
		return l
	}
	return nil
}
