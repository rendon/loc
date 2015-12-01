package main

import (
	"database/sql"
	"errors"
	"fmt"
	logger "log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/goutil/ds"
	"github.com/kellydunn/golang-geo"
	_ "github.com/mattn/go-sqlite3"
)

type Location struct {
	Continent        string
	Country          string
	ShortCountryCode string
	LongCountryCode  string
	City             string
	Address          string
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

	// Simple floating point regex
	r  = `[-+]?[0-9]+(\.[0-9]*)?`
	re = regexp.MustCompile(fmt.Sprintf(`%s\s*,\s*%s`, r, r))

	geocoder = geo.GoogleGeocoder{}
)

func init() {
	log = logger.New(os.Stderr, "", 0)
	// You'll need a Google API key.
	geo.SetGoogleAPIKey(os.Getenv("GOOGLE_GEO_API_KEY"))
}

func parseCoordinate(c string) (*geo.Point, error) {
	c = strings.Replace(c, " ", "", -1)
	points := strings.Split(c, ",")
	if len(points) != 2 {
		return nil, errors.New("Invalid coordinates")
	}
	lat, err := strconv.ParseFloat(points[0], 64)
	if err != nil {
		return nil, err
	}
	long, err := strconv.ParseFloat(points[1], 64)
	if err != nil {
		return nil, err
	}
	return geo.NewPoint(lat, long), nil
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

func cleanString(str string) string {
	str = strings.ToLower(str)
	str = strings.Trim(str, " ")
	str = strings.Replace(str, "á", "a", -1)
	str = strings.Replace(str, "é", "e", -1)
	str = strings.Replace(str, "í", "i", -1)
	str = strings.Replace(str, "ó", "o", -1)
	str = strings.Replace(str, "ú", "u", -1)
	str = punctRe.ReplaceAllString(str, "")
	str = specialRe.ReplaceAllString(str, "")
	return str
}

func normalizeLocation(loc string) *Location {
	if match := re.FindString(loc); match != "" {
		p, err := parseCoordinate(match)
		if err != nil {
			log.Println(err)
			return nil
		}

		addr, err := geocoder.ReverseGeocode(p)
		if err != nil {
			log.Println(err)
			return nil
		}

		tokens := strings.Split(addr, ",")
		country := strings.ToLower(tokens[len(tokens)-1])
		if l, ok := countryTrie.Find(country).(*Location); ok && l != nil {
			l.Address = addr
			return l
		}

		if l := countryCodes[country]; l != nil {
			l.Address = addr
			return l
		}
	}

	loc = cleanString(loc)
	tokens := splitRe.Split(loc, -1)
	for i := 0; i < len(tokens); i++ {
		tokens[i] = cleanString(tokens[i])
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
