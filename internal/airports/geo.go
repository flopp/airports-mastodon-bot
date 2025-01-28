package airports

import (
	"fmt"
	"math"
	"strconv"
)

type LatLon struct {
	Lat float64
	Lon float64
}

func (ll LatLon) IsValid() bool {
	return ll.Lat >= -90 && ll.Lat <= 90 && ll.Lon >= -180 && ll.Lon <= 180
}

func ParseLatLon(lats, lons string) (LatLon, error) {
	lat, err := strconv.ParseFloat(lats, 64)
	if err != nil {
		return LatLon{1000, 1000}, fmt.Errorf("cannot parse latitude '%s': %w", lats, err)
	}
	if lat < -90 || lat > 90 {
		return LatLon{1000, 1000}, fmt.Errorf("invalid latitude '%s': %w", lats, err)
	}
	lon, err := strconv.ParseFloat(lons, 64)
	if err != nil {
		return LatLon{1000, 1000}, fmt.Errorf("cannot parse longitide '%s': %w", lons, err)
	}
	if lon < -180 || lon > 180 {
		return LatLon{1000, 1000}, fmt.Errorf("invalid longitude '%s': %w", lons, err)
	}

	return LatLon{Lat: lat, Lon: lon}, nil
}

func deg2rad(d float64) float64 {
	return d * math.Pi / 180.0
}

func rad2deg(r float64) float64 {
	return r * 180.0 / math.Pi
}

func normalizeAngle(deg float64) float64 {
	deg = math.Mod(deg, 360.0)
	if deg < 0 {
		deg += 360.0
	}
	return deg
}

func (latlon1 LatLon) DistanceBearing(latlon2 LatLon) (float64, float64, error) {
	if !latlon1.IsValid() || !latlon2.IsValid() {
		return 0, 0, fmt.Errorf("cannot compute distance/bearing for invalid latlons")
	}

	const earthRadiusKM float64 = 6371.0

	lat1 := deg2rad(latlon1.Lat)
	lon1 := deg2rad(latlon1.Lon)
	lat2 := deg2rad(latlon2.Lat)
	lon2 := deg2rad(latlon2.Lon)

	dlat := lat2 - lat1
	dlon := lon2 - lon1

	a := math.Pow(math.Sin(dlat/2), 2) + math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(dlon/2), 2)
	distance := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a)) * earthRadiusKM

	y := math.Sin(dlon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) -
		math.Sin(lat1)*math.Cos(lat2)*math.Cos(dlon)
	t := math.Atan2(y, x)

	bearing := normalizeAngle(rad2deg(t))
	return distance, bearing, nil
}
