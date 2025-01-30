package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/flopp/airports-mastodon-bot/internal/airports"
	"github.com/flopp/airports-mastodon-bot/internal/data"
	sm "github.com/flopp/go-staticmaps"
	"github.com/fogleman/gg"
	"github.com/golang/geo/s2"
)

const (
	usage = `USAGE: %s [OPTIONS...]
Run the airports-mastodon-bot cli. 

OPTIONS:
`
)

type Options struct {
	DataPath string
	Airport  string
}

func parseCommandLine() Options {
	data := flag.String("data", ".data", "data folder")
	airport := flag.String("airport", "", "ICAO code of airport to render")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(flag.Args()) != 0 {
		fmt.Println("ERROR: invalid command line")
		flag.Usage()
		os.Exit(1)
	}

	return Options{*data, *airport}
}

func is_interesting(airport *airports.Airport) bool {
	minRunwayLength := 1.0
	interestingRunways := make([]*airports.Runway, 0, len(airport.Runways))
	for _, runway := range airport.Runways {
		if d, _, err := runway.LeLatLon.DistanceBearing(runway.HeLatLon); err != nil || d < minRunwayLength {
			continue
		}
		interestingRunways = append(interestingRunways, runway)
	}

	if airport.Type == "large_airport" {
		if len(interestingRunways) < 1 {
			return false
		}
	} else if airport.Type == "medium_airport" {
		if len(interestingRunways) < 3 {
			return false
		}
	} else {
		return false
	}

	return true
}

func createBbox(airport *airports.Airport, margin float64) (*s2.Rect, error) {
	bb := airport.GetBoundingBox(margin)
	return sm.CreateBBox(bb.Max.Lat, bb.Min.Lon, bb.Min.Lat, bb.Max.Lon)
}

func textAirport(airport *airports.Airport) {
	fmt.Printf("%s - %s, %s\n", airport.Name, airport.City, airport.Country.Name)
	fmt.Println("")

	if airport.Wikipedia != "" {
		fmt.Println(airport.Wikipedia)
	}
	fmt.Printf("https://www.openstreetmap.org/#map=13/%.6f/%.6f\n", airport.LatLon.Lat, airport.LatLon.Lon)
	fmt.Println("")

	tags := make([]string, 0, 4)
	if airport.ICAO != "" {
		tags = append(tags, fmt.Sprintf("#%s", airport.ICAO))
	}
	if airport.IATA != "" {
		tags = append(tags, fmt.Sprintf("#%s", airport.IATA))
	}
	tags = append(tags, fmt.Sprintf("#%s", data.SanitizeName(airport.City)))
	tags = append(tags, "#airport")

	fmt.Println(strings.Join(tags, " "))
}

func drawAirport(airport *airports.Airport, tiles *sm.TileProvider, path string) error {
	ctx := sm.NewContext()
	ctx.SetSize(1024, 1024)
	ctx.SetTileProvider(tiles)
	//ctx.OverrideAttribution("(C) Arcgis World Imagery")

	bbox, err := createBbox(airport, 0.002)
	if err != nil {
		return err
	}
	ctx.SetBoundingBox(*bbox)

	/*
		ctx.AddObject(
			sm.NewMarker(
				s2.LatLngFromDegrees(airport.LatLon.Lat, airport.LatLon.Lon),
				color.RGBA{0xff, 0, 0, 0xff},
				16.0,
			),
		)
		for _, runway := range airport.Runways {
			if runway.LeLatLon.IsValid() && runway.HeLatLon.IsValid() {
				le := s2.LatLngFromDegrees(runway.LeLatLon.Lat, runway.LeLatLon.Lon)
				he := s2.LatLngFromDegrees(runway.HeLatLon.Lat, runway.HeLatLon.Lon)
				ctx.AddObject(sm.NewMarker(
					le,
					color.RGBA{0, 0xff, 0, 0xff},
					8.0,
				))
				ctx.AddObject(sm.NewMarker(
					he,
					color.RGBA{0xff, 0, 0, 0xff},
					8.0,
				))
				path := make([]s2.LatLng, 2)
				path[0] = le
				path[1] = he
				ctx.AddObject(sm.NewPath(path, color.RGBA{0xff, 0, 0, 0xff}, 1))
			} else if runway.LeLatLon.IsValid() {
				le := s2.LatLngFromDegrees(runway.LeLatLon.Lat, runway.LeLatLon.Lon)
				ctx.AddObject(sm.NewMarker(
					le,
					color.RGBA{0, 0, 0xff, 0xff},
					8.0,
				))
			} else if runway.LeLatLon.IsValid() {
				he := s2.LatLngFromDegrees(runway.HeLatLon.Lat, runway.HeLatLon.Lon)
				ctx.AddObject(sm.NewMarker(
					he,
					color.RGBA{0, 0, 0xff, 0xff},
					8.0,
				))
			}
		}
	*/
	img, err := ctx.Render()
	if err != nil {
		return err
	}

	if err := gg.SaveJPG(path, img, 95); err != nil {
		return err
	}

	return nil
}

func main() {
	options := parseCommandLine()

	airports_csv := fmt.Sprintf("%s/airports.csv", options.DataPath)
	runways_csv := fmt.Sprintf("%s/runways.csv", options.DataPath)
	countries_csv := fmt.Sprintf("%s/countries.csv", options.DataPath)

	airports_data, err := data.ReadCsvFile(airports_csv)
	if err != nil {
		panic(err)
	}
	runways_data, err := data.ReadCsvFile(runways_csv)
	if err != nil {
		panic(err)
	}
	_ = runways_data

	countries_data, err := data.ReadCsvFile(countries_csv)
	if err != nil {
		panic(err)
	}

	countries_by_code := make(map[string]*airports.Country)
	for line, data := range countries_data {
		if line == 0 {
			continue
		}
		country, err := airports.CreateCountryFromCsvData(data)
		if err != nil {
			panic(fmt.Errorf("%s:%d could not parse airport: %w", countries_csv, line, err))
		}

		if existing_country, found := countries_by_code[country.Code]; found {
			panic(fmt.Errorf("counties with same code '%s': '%s', '%s'", country.Code, existing_country.Name, country.Name))
		}
		countries_by_code[country.Code] = &country
	}
	airports_by_icao := make(map[string]*airports.Airport)

	for line, data := range airports_data {
		if line == 0 {
			continue
		}
		airport, err := airports.CreateAiportFromCsvData(data, countries_by_code)
		if err != nil {
			panic(fmt.Errorf("%s:%d could not parse airport: %w", airports_csv, line, err))
		}

		if existing_airport, found := airports_by_icao[airport.ICAO]; found {
			panic(fmt.Errorf("airports with same ICAO code '%s': '%s', '%s'", airport.ICAO, existing_airport.Name, airport.Name))
		}
		airports_by_icao[airport.ICAO] = &airport
	}
	fmt.Printf("found %d airports\n", len(airports_by_icao))

	runways_list := make([]*airports.Runway, 0, len(airports_by_icao))
	for line, data := range runways_data {
		if line == 0 {
			continue
		}
		runway, err := airports.CreateRunwayFromCsvData(data)
		if err != nil {
			panic(fmt.Errorf("%s:%d could not parse runway: %w", runways_csv, line, err))
		}

		runways_list = append(runways_list, &runway)

		if airport, found := airports_by_icao[runway.AirportICAO]; found {
			airport.Runways = append(airport.Runways, &runway)
		} else {
			panic(fmt.Errorf("%s:%d cannot find airport by ICAO '%s", runways_csv, line, runway.AirportICAO))
		}
	}
	fmt.Printf("found %d runways\n", len(runways_list))

	interesting_airports := make([]*airports.Airport, 0)
	for _, airport := range airports_by_icao {
		if is_interesting(airport) {
			interesting_airports = append(interesting_airports, airport)
		}
	}
	fmt.Printf("interesting airports: %d\n", len(interesting_airports))

	tilesOSM := sm.NewTileProviderOpenStreetMaps()
	tilesAerial := sm.NewTileProviderArcgisWorldImagery()

	if options.Airport != "" {
		airport, found := airports_by_icao[options.Airport]
		if !found {
			fmt.Printf("Cannot find airport by ICAO '%s'\n", options.Airport)
			os.Exit(1)
		}

		textAirport(airport)
		if err := drawAirport(airport, tilesAerial, fmt.Sprintf("render/%s-aerial.jpg", airport.ICAO)); err != nil {
			panic(fmt.Errorf("cannot draw airport '%s': %w", airport.Name, err))
		}
		if err := drawAirport(airport, tilesOSM, fmt.Sprintf("render/%s-osm.jpg", airport.ICAO)); err != nil {
			panic(fmt.Errorf("cannot draw airport '%s': %w", airport.Name, err))
		}
	} else {
		for _, airport := range interesting_airports {
			textAirport(airport)
			if err := drawAirport(airport, tilesAerial, fmt.Sprintf("render/%s-aerial.jpg", airport.ICAO)); err != nil {
				panic(fmt.Errorf("cannot draw airport '%s': %w", airport.Name, err))
			}
			if err := drawAirport(airport, tilesOSM, fmt.Sprintf("render/%s-osm.jpg", airport.ICAO)); err != nil {
				panic(fmt.Errorf("cannot draw airport '%s': %w", airport.Name, err))
			}
			fmt.Println()
			fmt.Println()
		}
	}
}
