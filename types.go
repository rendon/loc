package loc

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
	Guesses           []string `json:"guesses"`
	ShortCode         string   `json:"short_code"`
	LongCode          string   `json:"long_code"`
}
