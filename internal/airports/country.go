package airports

import (
	"fmt"
)

type Country struct {
	Code string
	Name string
}

func CreateCountryFromCsvData(data []string) (Country, error) {
	if len(data) != 6 {
		return Country{}, fmt.Errorf("expected 6 data items, got %d", len(data))
	}

	// id := data[0]
	code := data[1]
	name := data[2]
	// continent := data[3]
	// wikipedia := data[4]
	// keywords := data[5]

	return Country{Code: code, Name: name}, nil
}
