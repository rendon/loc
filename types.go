package loc

type Location struct {
	Continent        string `json:"continent"`
	Country          string `json:"country"`
	ShortCountryCode string `json:"short_country_code"`
	LongCountryCode  string `json:"long_country_code"`
	City             string `json:"city"`
	Address          string `json:"address"`
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
