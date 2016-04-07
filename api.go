package loc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/goutil/ds"
	"github.com/kellydunn/golang-geo"
	_ "github.com/mattn/go-sqlite3"
)

var (
	splitRe   = regexp.MustCompile(`\s*[,|/-]\s*`)
	punctRe   = regexp.MustCompile("[.;:]")
	accentRe  = regexp.MustCompile("[áéíóú`]")
	specialRe = regexp.MustCompile(`[♥✈️]'"`)

	countryTrie = ds.NewTrie()
	cityTrie    = ds.NewTrie()
	abbrTrie    = ds.NewTrie()
	guessTrie   = ds.NewTrie()

	countryCodes = make(map[string]*Location)

	// Simple floating point regex
	r        = `[-+]?[0-9]+(\.[0-9]*)?`
	re       = regexp.MustCompile(fmt.Sprintf(`%s\s*,\s*%s`, r, r))
	geocoder = geo.GoogleGeocoder{}

	initialized = false
)

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

func Initialize() {
	// You'll need a Google API key.
	geo.SetGoogleAPIKey(os.Getenv("GOOGLE_GEO_API_KEY"))
	locDB := os.Getenv("LOC_DB")
	buf, err := ioutil.ReadFile(locDB)
	if err != nil {
		log.Fatalf("Failed to load location database: %s", err)
	}

	var countries []Country
	if err := json.Unmarshal(buf, &countries); err != nil {
		log.Fatalf("Failed to unmarshal countries: %s\n", err)
	}

	for _, country := range countries {
		country.Name = strings.ToLower(country.Name)
		country.ShortCode = strings.ToLower(country.ShortCode)
		country.LongCode = strings.ToLower(country.LongCode)
		loc := Location{
			Country:          country.Name,
			ShortCountryCode: country.ShortCode,
			LongCountryCode:  country.LongCode,
		}
		countryCodes[loc.ShortCountryCode] = &loc
		countryCodes[loc.LongCountryCode] = &loc
		countryTrie.Insert(country.Name, &loc)
		for _, name := range country.Names {
			countryTrie.Insert(strings.ToLower(name), &loc)
		}
		for _, c := range country.Cities {
			cityTrie.Insert(cleanString(c), &loc)
		}
		for _, c := range country.CityAbbreviations {
			abbrTrie.Insert(cleanString(c), &loc)
		}
		for _, c := range country.Guesses {
			guessTrie.Insert(cleanString(c), &loc)
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

func findLocationByCoordinates(loc string) *Location {
	p, err := parseCoordinate(loc)
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
	return nil
}

func Locate(loc string) *Location {
	if !initialized {
		Initialize()
		initialized = true
	}
	if match := re.FindString(loc); match != "" {
		return findLocationByCoordinates(match)
	}

	loc = cleanString(loc)
	tokens := splitRe.Split(loc, -1)
	for i := 0; i < len(tokens); i++ {
		tokens[i] = cleanString(tokens[i])
	}

	// Exact match with country
	l, ok := countryTrie.Find(loc).(*Location)
	if ok && l != nil {
		return l
	}

	// Exact match with city
	l, ok = cityTrie.Find(loc).(*Location)
	if ok && l != nil {
		return l
	}

	// city, country OR country, city
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

	// By country code
	if len(tokens) == 2 {
		if countryCodes[tokens[1]] != nil {
			return countryCodes[tokens[1]]
		}
	}

	// Brute force...
	// Trie all possible substrings, `loc` is expected to be a short string
	size := len(loc)
	// By country
	for s := size; s >= 4; s-- {
		for i := 0; i+s <= size; i++ {
			ss := loc[i : i+s]
			if l, ok := countryTrie.Find(ss).(*Location); ok && l != nil {
				return l
			}
		}
	}

	// By city
	for s := size; s >= 4; s-- {
		for i := 0; i+s <= size; i++ {
			ss := loc[i : i+s]
			if l, ok := cityTrie.Find(ss).(*Location); ok && l != nil {
				return l
			}
		}
	}
	// By abbreviation
	for i := 0; i < len(tokens); i++ {
		if l, ok := abbrTrie.Find(tokens[i]).(*Location); ok && l != nil {
			return l
		}
	}

	// And finally... some guessing
	for s := size; s >= 2; s-- {
		for i := 0; i+s <= size; i++ {
			ss := loc[i : i+s]
			if l, ok := guessTrie.Find(ss).(*Location); ok && l != nil {
				return l
			}
		}
	}
	return nil
}
