package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"

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

type Item struct {
	Name   string    `json:"name"`
	Vector []float64 `json:"vector"`
}

type Data struct {
	Name    string `json:"name"`
	Vectors []Item `json:"embeddings"`
}

var (
	swiftPath string
	mpPath    string
	title     string
	dim       int
	gradient  bool
	out       string
)

func init() {
	flag.StringVar(&swiftPath, "swift-path", "", "taylor swift embeddings path")
	flag.StringVar(&mpPath, "mp-path", "", "masterplar embeddings path")
	flag.IntVar(&dim, "dim", 2, "chart dimension used for plotting and PCA projection (2 or 3)")
	flag.BoolVar(&gradient, "gradient", false, "use gradient when coloring charts")
	flag.StringVar(&title, "title", "Taylor Swift vs Masterplan", "title for the embeddings chart")
	flag.StringVar(&out, "out", "embeddings.html", "chart output path")
}

func getPCA(path string, dim int) ([]Data, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	var embs []Data
	if err := json.Unmarshal(b, &embs); err != nil {
		return nil, err
	}

	pcas := make([]Data, 0, len(embs))

	for _, e := range embs {
		items := make([]Item, 0, len(e.Vectors))
		// albumMx: each row is a song whose dimension is the length of its embedding
		albumMx := mat.NewDense(len(e.Vectors), len(e.Vectors[0].Vector), nil)
		for i, t := range e.Vectors {
			albumMx.SetRow(i, t.Vector)
		}
		r, _ := albumMx.Dims()
		if r == 1 {
			log.Printf("skipping data %s due to low number of items: %d", e.Name, len(e.Vectors))
			continue
		}
		var pc stat.PC
		ok := pc.PrincipalComponents(albumMx, nil)
		if !ok {
			log.Printf("failed pca for %s", e.Name)
			continue
		}
		var proj mat.Dense
		var vec mat.Dense
		pc.VectorsTo(&vec)
		proj.Mul(albumMx, vec.Slice(0, len(e.Vectors[0].Vector), 0, dim))

		for i := range e.Vectors {
			items = append(items, Item{
				Name:   e.Vectors[i].Name,
				Vector: proj.RawRowView(i),
			})
		}
		pcas = append(pcas, Data{
			Name:    e.Name,
			Vectors: items,
		})
	}

	return pcas, nil
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
	swiftPcas, err := getPCA(swiftPath, dim)
	if err != nil {
		log.Fatal(err)
	}

	// Masterplan data
	mpPcas, err := getPCA(mpPath, dim)
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
		add2DSeries(swift, swiftPcas, scatter)
		add2DSeries(mp, mpPcas, scatter)
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
