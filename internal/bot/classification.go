package bot

import "github.com/flopp/airports-mastodon-bot/internal/airports"

func IsInteresting(airport *airports.Airport) bool {
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
