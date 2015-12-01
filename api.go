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
	splitRe   = regexp.MustCompile(`\s*[,|/-]\s*`)
	punctRe   = regexp.MustCompile("[,.;:]")
	accentRe  = regexp.MustCompile("[áéíóú`]")
	specialRe = regexp.MustCompile(`[♥✈️]'"`)

	continentTrie = ds.NewTrie()
	countryTrie   = ds.NewTrie()
	cityTrie      = ds.NewTrie()
	mexicanCities = ds.NewTrie()

	countryCodes = make(map[string]*Location)
)

func init() {
	log = logger.New(os.Stderr, "", 0)
}

func initialize() {
	// Read world cities
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
		countryCodes[l.ShortCountryCode] = &l
		if l.ShortCountryCode == "MX" {
			mexicanCities.Insert(l.City, &l)
		}
	}
}

func normalizeLocation(loc string) *Location {
	loc = strings.ToLower(loc)
	loc = strings.Trim(loc, " ")
	loc = strings.Replace(loc, "á", "a", -1)
	loc = strings.Replace(loc, "é", "e", -1)
	loc = strings.Replace(loc, "í", "i", -1)
	loc = strings.Replace(loc, "ó", "o", -1)
	loc = strings.Replace(loc, "ú", "u", -1)
	tokens := splitRe.Split(loc, -1)
	for i := 0; i < len(tokens); i++ {
		tokens[i] = punctRe.ReplaceAllString(tokens[i], "")
		tokens[i] = specialRe.ReplaceAllString(tokens[i], "")
		tokens[i] = strings.Replace(tokens[i], "á", "a", -1)
		tokens[i] = strings.Replace(tokens[i], "é", "e", -1)
		tokens[i] = strings.Replace(tokens[i], "í", "i", -1)
		tokens[i] = strings.Replace(tokens[i], "ó", "o", -1)
		tokens[i] = strings.Replace(tokens[i], "ó", "u", -1)
	}

	// Case 1: ciy, country OR country, city
	if len(tokens) == 2 {
		if l, ok := countryTrie.Find(tokens[1]).(*Location); ok && l != nil {
			return l
		}
		if l, ok := countryTrie.Find(tokens[0]).(*Location); ok && l != nil {
			return l
		}

		// Very special case for México
		if l, ok := mexicanCities.Find(loc).(*Location); ok && l != nil {
			return l
		}

		for i := 0; i < len(tokens); i++ {
			if l, ok := mexicanCities.Find(tokens[i]).(*Location); ok && l != nil {
				return l
			}
		}

		if l, ok := cityTrie.Find(tokens[0]).(*Location); ok && l != nil {
			return l
		}
		if l, ok := cityTrie.Find(tokens[1]).(*Location); ok && l != nil {
			return l
		}
	}

	// Case 2a: Exact match with country
	l, ok := countryTrie.Find(loc).(*Location)
	if ok && l != nil {
		return l
	}

	// Case 2b: Exact match with city
	l, ok = cityTrie.Find(loc).(*Location)
	if ok && l != nil {
		return l
	}

	// Case 3: By country code
	if len(tokens) == 2 {
		if countryCodes[tokens[1]] != nil {
			return countryCodes[tokens[1]]
		}
	}

	// Case 4: Try all tokens by country, city and continent
	for _, t := range tokens {
		if l, ok = countryTrie.Find(t).(*Location); ok && l != nil {
			return l
		}
	}
	for _, t := range tokens {
		if l, ok = cityTrie.Find(t).(*Location); ok && l != nil {
			return l
		}
	}
	for _, t := range tokens {
		if l, ok = continentTrie.Find(t).(*Location); ok && l != nil {
			return l
		}
	}
	return nil
}
