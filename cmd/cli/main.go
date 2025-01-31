package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/flopp/airports-mastodon-bot/internal/airports"
	"github.com/flopp/airports-mastodon-bot/internal/bot"
	"github.com/flopp/airports-mastodon-bot/internal/data"
	sm "github.com/flopp/go-staticmaps"
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
	data, err := bot.DrawAirport(airport, tiles)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o777)
}

func main() {
	options := parseCommandLine()

	airports_csv := fmt.Sprintf("%s/airports.csv", options.DataPath)
	runways_csv := fmt.Sprintf("%s/runways.csv", options.DataPath)
	countries_csv := fmt.Sprintf("%s/countries.csv", options.DataPath)

	airports_by_icao, err := airports.Load(airports_csv, runways_csv, countries_csv)
	if err != nil {
		panic(err)
	}

	interesting_airports := make([]*airports.Airport, 0)
	for _, airport := range airports_by_icao {
		if bot.IsInteresting(airport) {
			interesting_airports = append(interesting_airports, airport)
		}
	}
	fmt.Printf("interesting airports: %d / %d\n", len(interesting_airports), len(airports_by_icao))

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
