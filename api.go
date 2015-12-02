package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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

type Country struct {
	Name              string   `json:"name"`
	Names             []string `json:"names"`
	Cities            []string `json:"cities"`
	CityAbbreviations []string `json:"city_abbreviations"`
	ShortCode         string   `json:"short_code"`
	LongCode          string   `json:"long_code"`
}

var (
	splitRe   = regexp.MustCompile(`\s*[,|/-]\s*`)
	punctRe   = regexp.MustCompile("[.;:]")
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
	// Read cities
	var countries []Country
	buf, err := ioutil.ReadFile("data/countries.json")
	if err != nil {
		log.Fatalf("Failed to load cities: %s\n", err)
	}
	if err := json.Unmarshal(buf, &countries); err != nil {
		log.Fatalf("Failed to unmarshal countries: %s\n", err)
	}

	for _, country := range countries {
		loc := Location{
			Country:          country.Name,
			ShortCountryCode: country.ShortCode,
			LongCountryCode:  country.LongCode,
		}
		countryTrie.Insert(country.Name, &loc)
		for _, name := range country.Names {
			countryTrie.Insert(strings.ToLower(name), &loc)
		}
		for _, c := range country.Cities {
			cityTrie.Insert(cleanString(c), &loc)
		}
		for _, c := range country.CityAbbreviations {
			cityTrie.Insert(cleanString(c), &loc)
		}
	}
	fmt.Println()
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

	// Case 1: city, country OR country, city
	if len(tokens) == 2 {
		if l, ok := countryTrie.Find(tokens[1]).(*Location); ok && l != nil {
			return l
		}
		if l, ok := countryTrie.Find(tokens[0]).(*Location); ok && l != nil {
			return l
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
