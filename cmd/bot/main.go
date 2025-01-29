package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image/jpeg"
	"os"
	"strings"
	"time"

	"github.com/flopp/airports-mastodon-bot/internal/airports"
	"github.com/flopp/airports-mastodon-bot/internal/data"
	sm "github.com/flopp/go-staticmaps"
	"github.com/golang/geo/s2"
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

func createBbox(airport *airports.Airport) (*s2.Rect, error) {
	minLat := airport.LatLon.Lat
	maxLat := minLat
	minLon := airport.LatLon.Lon
	maxLon := minLon

	for _, runway := range airport.Runways {
		if runway.LeLatLon.IsValid() {
			if runway.LeLatLon.Lat < minLat {
				minLat = runway.LeLatLon.Lat
			} else if runway.LeLatLon.Lat > maxLat {
				maxLat = runway.LeLatLon.Lat
			}

			if runway.LeLatLon.Lon < minLon {
				minLon = runway.LeLatLon.Lon
			} else if runway.LeLatLon.Lon > maxLon {
				maxLon = runway.LeLatLon.Lon
			}
		}

		if runway.HeLatLon.IsValid() {
			if runway.HeLatLon.Lat < minLat {
				minLat = runway.HeLatLon.Lat
			} else if runway.HeLatLon.Lat > maxLat {
				maxLat = runway.HeLatLon.Lat
			}

			if runway.HeLatLon.Lon < minLon {
				minLon = runway.HeLatLon.Lon
			} else if runway.HeLatLon.Lon > maxLon {
				maxLon = runway.HeLatLon.Lon
			}
		}
	}

	return sm.CreateBBox(maxLat, minLon, minLat, maxLon)
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

func drawAirport(airport *airports.Airport, tiles *sm.TileProvider) ([]byte, error) {
	ctx := sm.NewContext()
	ctx.SetSize(1024, 1024)
	ctx.SetTileProvider(tiles)

	bbox, err := createBbox(airport)
	if err != nil {
		return nil, err
	}
	ctx.SetBoundingBox(*bbox)

	img, err := ctx.Render()
	if err != nil {
		return nil, err
	}

	buff := new(bytes.Buffer)
	var byteWriter = bufio.NewWriter(buff)
	if err := jpeg.Encode(byteWriter, img, nil); err != nil {
		return nil, fmt.Errorf("failed to encode jpg: %w", err)
	}

	return buff.Bytes(), nil
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

	airports_data, err := data.ReadCsvFile(airports_csv)
	if err != nil {
		panic(err)
	}
	runways_data, err := data.ReadCsvFile(runways_csv)
	if err != nil {
		panic(err)
	}
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

	for line, data := range runways_data {
		if line == 0 {
			continue
		}
		runway, err := airports.CreateRunwayFromCsvData(data)
		if err != nil {
			panic(fmt.Errorf("%s:%d could not parse runway: %w", runways_csv, line, err))
		}

		if airport, found := airports_by_icao[runway.AirportICAO]; found {
			airport.Runways = append(airport.Runways, &runway)
		} else {
			panic(fmt.Errorf("%s:%d cannot find airport by ICAO '%s", runways_csv, line, runway.AirportICAO))
		}
	}

	interesting_airports := make([]*airports.Airport, 0)
	for _, airport := range airports_by_icao {
		if is_interesting(airport) {
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

	aerialImage, err := drawAirport(airport, tilesAerial)
	if err != nil {
		panic(fmt.Errorf("cannot draw aerial image of airport '%s': %w", airport.Name, err))
	}

	osmImage, err := drawAirport(airport, tilesOSM)
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
