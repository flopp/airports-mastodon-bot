package airports

import (
	"fmt"
	"strconv"
	"strings"
)

type Runway struct {
	AirportICAO string
	Surface     string
	LengthFt    float64
	WidthFt     float64
	LeLatLon    LatLon
	HeLatLon    LatLon
}

func CreateRunwayFromCsvData(data []string) (Runway, error) {
	if len(data) != 20 {
		return Runway{}, fmt.Errorf("expected 20 data items, got %d", len(data))
	}

	// _id := data[0]
	// _airport_ref := data[1]
	airport_icao := strings.ToUpper(data[2])
	s_length_ft := data[3]
	s_width_ft := data[4]
	surface := data[5]
	// _lighted := data[6]
	// _closed := data[7]
	// _le_ident := data[8].replace('"', "")
	s_le_lat := data[9]
	s_le_lon := data[10]
	// _le_elevation_ft := data[11]
	// _le_heading_degT := data[12]
	// _le_displaced_threshold_ft := data[14]
	// _he_ident := data[14].replace('"', "")
	s_he_lat := data[15]
	s_he_lon := data[16]
	// _he_elevation_ft := data[17]
	// _he_heading_degT := data[18]
	// _he_displaced_threshold_ft := data[19]

	length_ft := 0.0
	if s_length_ft != "" {
		if v, err := strconv.ParseFloat(s_length_ft, 64); err != nil {
			return Runway{}, fmt.Errorf("failed to parse length '%s': %w", s_length_ft, err)
		} else {
			length_ft = v
		}
	}

	width_ft := 0.0
	if s_width_ft != "" {
		if v, err := strconv.ParseFloat(s_width_ft, 64); err != nil {
			return Runway{}, fmt.Errorf("failed to parse width '%s': %w", s_width_ft, err)
		} else {
			width_ft = v
		}
	}

	le_latlon := LatLon{1000, 1000}
	if s_le_lat != "" && s_le_lon != "" {
		if latlon, err := ParseLatLon(s_le_lat, s_le_lon); err != nil {
			return Runway{}, fmt.Errorf("failed to parse lat/lon coordinates of LE: %w", err)
		} else {
			le_latlon = latlon
		}
	}

	he_latlon := LatLon{1000, 1000}
	if s_he_lat != "" && s_he_lon != "" {
		if latlon, err := ParseLatLon(s_he_lat, s_he_lon); err != nil {
			return Runway{}, fmt.Errorf("failed to parse lat/lon coordinates of HE: %w", err)
		} else {
			he_latlon = latlon
		}
	}

	return Runway{AirportICAO: airport_icao, Surface: surface, LengthFt: length_ft, WidthFt: width_ft, LeLatLon: le_latlon, HeLatLon: he_latlon}, nil
}
