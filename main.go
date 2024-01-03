package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"strings"

	"github.com/danaugrs/go-tsne/tsne"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/render"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

const (
	swift = "Taylor Swift"
	mp    = "Masterplan"
)

type Vector struct {
	Name   string    `json:"name"`
	Values []float64 `json:"vector"`
}

type Data struct {
	Name    string   `json:"name"`
	Vectors []Vector `json:"embeddings"`
}

var (
	swiftPath string
	mpPath    string
	title     string
	dim       int
	gradient  bool
	out       string
	proj      string
)

func init() {
	flag.StringVar(&swiftPath, "swift-path", "", "taylor swift embeddings path")
	flag.StringVar(&mpPath, "mp-path", "", "masterplar embeddings path")
	flag.IntVar(&dim, "dim", 2, "chart dimension used for plotting and data projection (2 or 3)")
	flag.BoolVar(&gradient, "gradient", false, "use gradient when coloring charts")
	flag.StringVar(&title, "title", "Taylor Swift vs Masterplan", "title for the embeddings chart")
	flag.StringVar(&out, "out", "embeddings.html", "chart output path")
	flag.StringVar(&proj, "proj", "pca", "projection (pca or tsne)")
}

func getTSNE(embs []Data, dim int) ([]Data, error) {
	tsnes := make([]Data, 0, len(embs))

	perplexity, learningRate := float64(30), float64(200)
	if dim == 3 {
		perplexity, learningRate = float64(30), float64(200)
	}

	for _, e := range embs {
		items := make([]Vector, 0, len(e.Vectors))
		// embMx: each row is a song whose dimension is the length of its embedding
		embMx := mat.NewDense(len(e.Vectors), len(e.Vectors[0].Values), nil)
		for i, t := range e.Vectors {
			embMx.SetRow(i, t.Values)
		}

		t := tsne.NewTSNE(dim, perplexity, learningRate, 3000, true)
		resMat := t.EmbedData(embMx, nil)
		d := mat.DenseCopyOf(resMat)

		for i := range e.Vectors {
			items = append(items, Vector{
				Name:   e.Vectors[i].Name,
				Values: d.RawRowView(i),
			})
		}
		tsnes = append(tsnes, Data{
			Name:    e.Name,
			Vectors: items,
		})
	}

	return tsnes, nil
}

func getPCA(embs []Data, dim int) ([]Data, error) {
	pcas := make([]Data, 0, len(embs))

	for _, e := range embs {
		items := make([]Vector, 0, len(e.Vectors))
		// embMx: each row is a song whose dimension is the length of its embedding
		embMx := mat.NewDense(len(e.Vectors), len(e.Vectors[0].Values), nil)
		for i, t := range e.Vectors {
			embMx.SetRow(i, t.Values)
		}
		r, _ := embMx.Dims()
		if r == 1 {
			log.Printf("skipping %s: low number of items: %d", e.Name, len(e.Vectors))
			continue
		}
		var pc stat.PC
		ok := pc.PrincipalComponents(embMx, nil)
		if !ok {
			log.Printf("failed pca for %s", e.Name)
			continue
		}
		var proj mat.Dense
		var vec mat.Dense
		pc.VectorsTo(&vec)
		proj.Mul(embMx, vec.Slice(0, len(e.Vectors[0].Values), 0, dim))

		for i := range e.Vectors {
			items = append(items, Vector{
				Name:   e.Vectors[i].Name,
				Values: proj.RawRowView(i),
			})
		}
		pcas = append(pcas, Data{
			Name:    e.Name,
			Vectors: items,
		})
	}

	return pcas, nil
}

func getSeriesData(path string, proj string, dim int) ([]Data, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	var embs []Data
	if err := json.Unmarshal(b, &embs); err != nil {
		return nil, err
	}

	if strings.EqualFold(proj, "tsne") {
		return getTSNE(embs, dim)
	}

	return getPCA(embs, dim)
}

func main() {
	flag.Parse()

	if swiftPath == "" || mpPath == "" {
		log.Fatal("empty path provided")
	}

	if dim <= 1 || dim > 3 {
		log.Fatal("invalid chart dimension")
	}

	// Taylor Swift data
	swiftPcas, err := getSeriesData(swiftPath, proj, dim)
	if err != nil {
		log.Fatal(err)
	}

	// Masterplan data
	mpPcas, err := getSeriesData(mpPath, proj, dim)
	if err != nil {
		log.Fatal(err)
	}

	// global options
	chartOptions := []charts.GlobalOpts{
		charts.WithTitleOpts(opts.Title{
			Title:    title,
			Subtitle: "Lyrics Embeddings",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:      true,
			Formatter: "{a}: {b}",
		}),
		charts.WithToolboxOpts(opts.Toolbox{
			Show:   true,
			Orient: "horizontal",
			Left:   "right",
			Feature: &opts.ToolBoxFeature{
				SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
					Show: true, Title: "Save as image"},
				Restore: &opts.ToolBoxFeatureRestore{
					Show: true, Title: "Reset"},
			}}),
	}

	var r render.Renderer

	switch dim {
	case 2:
		scatter := charts.NewScatter()
		scatter.SetGlobalOptions(chartOptions...)
		if err := add2DSeries(swift, swiftPcas, scatter); err != nil {
			log.Fatal(err)
		}
		if err := add2DSeries(mp, mpPcas, scatter); err != nil {
			log.Fatal(err)
		}
		r = scatter
	case 3:
		scatter3d := charts.NewScatter3D()
		scatter3d.SetGlobalOptions(chartOptions...)
		if err := add3DSeries(swift, swiftPcas, scatter3d, gradient); err != nil {
			log.Fatal(err)
		}
		if err := add3DSeries(mp, mpPcas, scatter3d, gradient); err != nil {
			log.Fatal(err)
		}
		r = scatter3d
	}

	f, err := os.Create(out)
	if err != nil {
		panic(err)
	}
	if err := r.Render(io.MultiWriter(f)); err != nil {
		log.Fatal(err)
	}
}
