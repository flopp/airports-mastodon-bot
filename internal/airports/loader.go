package airports

import (
	"fmt"

	"github.com/flopp/airports-mastodon-bot/internal/data"
)

func Load(airports_file, runways_file, countries_file string) (map[string]*Airport, error) {

	airports_data, err := data.ReadCsvFile(airports_file)
	if err != nil {
		panic(err)
	}
	runways_data, err := data.ReadCsvFile(runways_file)
	if err != nil {
		panic(err)
	}
	countries_data, err := data.ReadCsvFile(countries_file)
	if err != nil {
		panic(err)
	}

	countries_by_code := make(map[string]*Country)
	for line, data := range countries_data {
		if line == 0 {
			continue
		}
		country, err := CreateCountryFromCsvData(data)
		if err != nil {
			return nil, (fmt.Errorf("%s:%d could not parse airport: %w", countries_file, line, err))
		}

		if existing_country, found := countries_by_code[country.Code]; found {
			return nil, (fmt.Errorf("counties with same code '%s': '%s', '%s'", country.Code, existing_country.Name, country.Name))
		}
		countries_by_code[country.Code] = &country
	}

	airports_by_icao := make(map[string]*Airport)

	for line, data := range airports_data {
		if line == 0 {
			continue
		}
		airport, err := CreateAiportFromCsvData(data, countries_by_code)
		if err != nil {
			return nil, (fmt.Errorf("%s:%d could not parse airport: %w", airports_file, line, err))
		}

		if existing_airport, found := airports_by_icao[airport.ICAO]; found {
			return nil, (fmt.Errorf("airports with same ICAO code '%s': '%s', '%s'", airport.ICAO, existing_airport.Name, airport.Name))
		}
		airports_by_icao[airport.ICAO] = &airport
	}

	for line, data := range runways_data {
		if line == 0 {
			continue
		}
		runway, err := CreateRunwayFromCsvData(data)
		if err != nil {
			return nil, (fmt.Errorf("%s:%d could not parse runway: %w", runways_file, line, err))
		}

		if airport, found := airports_by_icao[runway.AirportICAO]; found {
			airport.Runways = append(airport.Runways, &runway)
		} else {
			return nil, (fmt.Errorf("%s:%d cannot find airport by ICAO '%s", runways_file, line, runway.AirportICAO))
		}
	}

	return airports_by_icao, nil
}
