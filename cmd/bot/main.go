package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flopp/airports-mastodon-bot/internal/airports"
	"github.com/flopp/airports-mastodon-bot/internal/bot"
	"github.com/flopp/airports-mastodon-bot/internal/data"
	sm "github.com/flopp/go-staticmaps"
	"github.com/mattn/go-mastodon"
	"golang.org/x/exp/rand"
)

const (
	usage = `USAGE: %s [OPTIONS...]
Run the airports-mastodon-bot cli. 

OPTIONS:
`
)

type Options struct {
	DataPath   string
	ConfigPath string
}

func parseCommandLine() Options {
	data := flag.String("data", ".data", "data folder")
	config := flag.String("config", "production-config.json", "json file with mastodon config")

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

	return Options{*data, *config}
}

func readMastodonConfig(fileName string) (mastodon.Config, error) {
	confBytes, err := os.ReadFile(fileName)
	if err != nil {
		return mastodon.Config{}, fmt.Errorf("reading %s: %s", fileName, err)
	}

	var conf mastodon.Config
	if err = json.Unmarshal(confBytes, &conf); err != nil {
		return mastodon.Config{}, fmt.Errorf("unmarshalling %s: %s", fileName, err)
	}

	return conf, nil
}

func genMessage(airport *airports.Airport) string {
	msg := ""
	if airport.City != "" {
		msg += fmt.Sprintf("%s - %s, %s\n\n", airport.Name, airport.City, airport.Country.Name)
	} else {
		msg += fmt.Sprintf("%s - %s\n\n", airport.Name, airport.Country.Name)
	}

	if airport.Wikipedia != "" {
		msg += airport.Wikipedia
		msg += "\n"
	}
	msg += fmt.Sprintf("https://www.openstreetmap.org/#map=13/%.6f/%.6f\n\n", airport.LatLon.Lat, airport.LatLon.Lon)

	tags := make([]string, 0, 4)
	if airport.ICAO != "" {
		tags = append(tags, fmt.Sprintf("#%s", airport.ICAO))
	}
	if airport.IATA != "" {
		tags = append(tags, fmt.Sprintf("#%s", airport.IATA))
	}
	if airport.City != "" {
		tags = append(tags, fmt.Sprintf("#%s", data.SanitizeName(airport.City)))
	}
	tags = append(tags, fmt.Sprintf("#%s", data.SanitizeName(airport.Country.Name)))
	tags = append(tags, "#airport", "#aviation", "#avgeeks", "#GIS")

	msg += strings.Join(tags, " ")

	return msg
}

func main() {
	options := parseCommandLine()

	mastodonConfig, err := readMastodonConfig(options.ConfigPath)
	if err != nil {
		panic(fmt.Errorf("failed to read mastodon config: %w", err))
	}

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
			// double insertion of large airports to raise their chance
			if airport.Type == "large_airport" {
				interesting_airports = append(interesting_airports, airport)
			}
		}
	}

	tilesOSM := sm.NewTileProviderOpenStreetMaps()
	tilesAerial := sm.NewTileProviderArcgisWorldImagery()

	s := rand.NewSource(uint64(time.Now().Unix()))
	r := rand.New(s)
	airport := interesting_airports[r.Intn(len(interesting_airports))]

	client := mastodon.NewClient(&mastodonConfig)
	ctx := context.Background()

	toot := &mastodon.Toot{
		Status:     genMessage(airport),
		Visibility: "public",
		Language:   "en",
	}

	aerialImage, err := bot.DrawAirport(airport, tilesAerial)
	if err != nil {
		panic(fmt.Errorf("cannot draw aerial image of airport '%s': %w", airport.Name, err))
	}

	osmImage, err := bot.DrawAirport(airport, tilesOSM)
	if err != nil {
		panic(fmt.Errorf("cannot draw OSM image of airport '%s': %w", airport.Name, err))
	}

	if attachment, err := client.UploadMediaFromBytes(ctx, aerialImage); err == nil {
		toot.MediaIDs = append(toot.MediaIDs, attachment.ID)
	} else {
		panic(fmt.Errorf("cannot upload attachment: %w", err))
	}

	if attachment, err := client.UploadMediaFromBytes(ctx, osmImage); err == nil {
		toot.MediaIDs = append(toot.MediaIDs, attachment.ID)
	} else {
		panic(fmt.Errorf("cannot upload attachment: %w", err))
	}

	if _, err := client.PostStatus(context.Background(), toot); err != nil {
		panic(fmt.Errorf("failed to send status: %w", err))
	}
}
