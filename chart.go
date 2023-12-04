package main

import (
	"fmt"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/mazznoer/colorgrad"
)

const (
	swiftDefaultColor = "yellow"
	mpDefaultColor    = "black"
)

func getGradColors(gradient colorgrad.Gradient, count int) []string {
	gradColors := make([]string, count)
	for i, c := range gradient.ColorfulColors(uint(count)) {
		gradColors[i] = c.Hex()
	}
	return gradColors
}

func getDefaultColors(color string, count int) []string {
	colors := make([]string, count)
	for i := range colors {
		colors[i] = color
	}
	return colors
}

func getColors(artist string, grad bool, count int) ([]string, error) {
	switch artist {
	case swift:
		if grad {
			return getGradColors(colorgrad.YlOrRd(), count), nil

		}
		return getDefaultColors(swiftDefaultColor, count), nil
	case mp:
		if grad {
			colorGrad, err := colorgrad.NewGradient().Build()
			if err != nil {
				return nil, err
			}
			return getGradColors(colorGrad, count), nil
		}
		return getDefaultColors(mpDefaultColor, count), nil
	default:
		return nil, fmt.Errorf("unknown artist: %s", artist)
	}
}

func add2DSeries(artist string, data []Data, chart *charts.Scatter) {
	for _, d := range data {
		for _, p := range d.Vectors {
			vals := make([]interface{}, len(p.Vector))
			for i := range p.Vector {
				vals[i] = p.Vector[i]
			}
			fmt.Println("album: ", d.Name, "song:", p.Name)
			chart.AddSeries(artist, []opts.ScatterData{
				{
					Name:   fmt.Sprintf("%s (%s)", p.Name, d.Name),
					Value:  vals,
					Symbol: "roundRect",
				},
			})
		}
	}
}

func add3DSeries(artist string, data []Data, chart *charts.Scatter3D, grad bool) error {
	colors, err := getColors(artist, grad, len(data))
	if err != nil {
		return err
	}

	for i, d := range data {
		for _, p := range d.Vectors {
			vals := make([]interface{}, len(p.Vector))
			for i := range p.Vector {
				vals[i] = p.Vector[i]
			}
			chart.AddSeries(artist, []opts.Chart3DData{
				{
					Name:      fmt.Sprintf("%s (%s)", p.Name, d.Name),
					Value:     vals,
					ItemStyle: &opts.ItemStyle{Color: colors[i]},
					Label: &opts.Label{
						Show:      true,
						Formatter: "{a}: {b}",
					},
				},
			})
		}
	}
	return err
}
