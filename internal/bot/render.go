package bot

import (
	"bufio"
	"bytes"
	"fmt"
	"image/jpeg"

	"github.com/flopp/airports-mastodon-bot/internal/airports"
	sm "github.com/flopp/go-staticmaps"
	"github.com/golang/geo/s2"
)

func createBbox(airport *airports.Airport, margin float64) (*s2.Rect, error) {
	bb := airport.GetBoundingBox(margin)
	return sm.CreateBBox(bb.Max.Lat, bb.Min.Lon, bb.Min.Lat, bb.Max.Lon)
}

func DrawAirport(airport *airports.Airport, tiles *sm.TileProvider) ([]byte, error) {
	ctx := sm.NewContext()
	ctx.SetSize(1024, 1024)
	ctx.SetTileProvider(tiles)

	bbox, err := createBbox(airport, 0.002)
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
