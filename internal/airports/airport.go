package airports

import (
	"fmt"
	"strings"
)

type Airport struct {
	Type             string
	Name             string
	ICAO             string
	IATA             string
	Country          *Country
	City             string
	LatLon           LatLon
	Website          string
	Wikipedia        string
	Runways          []*Runway
	ExcessiveRunways []*Runway
}

func CreateAiportFromCsvData(data []string, countries map[string]*Country) (Airport, error) {
	if len(data) != 18 {
		return Airport{}, fmt.Errorf("expected 18 data items, got %d", len(data))
	}

	// 0 id
	icao := strings.ToUpper(data[1])
	type_ := data[2]
	name := data[3]
	lat := data[4]
	lon := data[5]
	// 6 eleveation_ft
	// 7 continent
	country_code := data[8]
	// 9 iso_region
	city := data[10]
	// 11 scheduled_service
	// 12 gps_code
	iata := strings.ToUpper(data[13])
	// 14 local_code
	website_url := data[15]
	wikipedia_url := data[16]
	// 17 keywords

	if iata == icao {
		iata = ""
	}

	country, found := countries[country_code]
	if !found {
		return Airport{}, fmt.Errorf("failed to fetch country: %s", country_code)
	}

	latlon, err := ParseLatLon(lat, lon)
	if err != nil {
		return Airport{}, fmt.Errorf("failed to parse lat/lon coordinates: %w", err)
	}

	return Airport{Type: type_, Name: name, ICAO: icao, IATA: iata, Country: country,
		City: city, LatLon: latlon, Website: website_url, Wikipedia: wikipedia_url,
		Runways: nil, ExcessiveRunways: nil}, nil
}

func (airport *Airport) AddRunway(runway *Runway) {
	excessiveDist := 20.0

	if !runway.LeLatLon.IsValid() && !runway.HeLatLon.IsValid() {
		airport.ExcessiveRunways = append(airport.ExcessiveRunways, runway)
	} else if d1, _, err := airport.LatLon.DistanceBearing(runway.LeLatLon); err == nil && d1 > excessiveDist {
		airport.ExcessiveRunways = append(airport.ExcessiveRunways, runway)
	} else if d2, _, err := airport.LatLon.DistanceBearing(runway.HeLatLon); err == nil && d2 > excessiveDist {
		airport.ExcessiveRunways = append(airport.ExcessiveRunways, runway)
	} else {
		airport.Runways = append(airport.Runways, runway)
	}
}
