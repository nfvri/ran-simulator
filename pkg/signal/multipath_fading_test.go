package signal

import (
	"fmt"
	"math"
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/nfvri/ran-simulator/pkg/model"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// PlotReceivedPower plots the received power values and saves it as a PNG file
func PlotReceivedPower(pathlossDb float64, realizations int, cell model.Cell) {
	receivedPowerDb := make(plotter.XYs, realizations)

	KdB := 9.0

	if cell.Channel.LOS {
		KdB = rand.NormFloat64()*RICEAN_K_STD_MACRO + RICEAN_K_MEAN
	}
	K := 10 * math.Log10(KdB)
	for i := 0; i < realizations; i++ {
		f := RiceanFading(K)
		if math.IsNaN(f) {
			continue
		}
		receivedPowerDb[i].X = float64(i)
		receivedPowerDb[i].Y = f
	}

	p := plot.New()
	p.Title.Text = fmt.Sprintf("LOS: %v", cell.Channel.LOS)
	p.X.Label.Text = "Realization"
	p.Y.Label.Text = "Received Power (dB)"

	line, points, err := plotter.NewLinePoints(receivedPowerDb)
	if err != nil {
		panic(err)
	}

	p.Add(line, points)
	p.Legend.Add("Received Power", line, points)

	// Create the output directory if it doesn't exist
	// if err := os.MkdirAll("/multipath_test_results/graphs/", os.ModePerm); err != nil {
	// 	panic(err)
	// }

	receivedPowerFilename := filepath.Join("./multipath_test_results/graphs/", fmt.Sprintf("%sLOS_%v.png", "multipath_db", cell.Channel.LOS))
	if err := p.Save(15*vg.Inch, 10*vg.Inch, receivedPowerFilename); err != nil {
		panic(err)
	}

	fmt.Printf("Plot saved to %s\n", receivedPowerFilename)

	// if err := os.MkdirAll("/multipath_test_results/distributions/", os.ModePerm); err != nil {
	// 	panic(err)
	// }

	// Plot the distribution separately
	distributionFilename := filepath.Join("./multipath_test_results/distributions/", fmt.Sprintf("%s_distribution_LOS_%v.png", "multipath_db_", cell.Channel.LOS))

	distribution := make(plotter.Values, len(receivedPowerDb))
	for i, pt := range receivedPowerDb {
		distribution[i] = pt.Y
	}

	pl := plot.New()
	pl.Title.Text = "Received Power Distribution"
	pl.X.Label.Text = "Received Power (dB)"
	pl.Y.Label.Text = "Frequency"

	h, err := plotter.NewHist(distribution, 50) // 50 bins for the histogram
	if err != nil {
		panic(err)
	}

	pl.Add(h)
	pl.Legend.Add("Received Power Distribution", h)

	if err := pl.Save(15*vg.Inch, 10*vg.Inch, distributionFilename); err != nil {
		panic(err)
	}

	fmt.Printf("Histogram plot saved to %s\n", distributionFilename)
}

func TestRayleighFading(t *testing.T) {
	cell := model.Cell{
		TxPowerDB: 45,
		Sector: model.Sector{
			Azimuth: 90,
			Center:  model.Coordinate{Lat: 37.979207, Lng: 23.716702},
			Height:  30,
		},
		Beam: model.Beam{
			H3dBAngle:              65,
			V3dBAngle:              65,
			MaxGain:                8,
			MaxAttenuationDB:       30,
			VSideLobeAttenuationDB: 30,
		},
		Channel: model.Channel{
			SSBFrequency: 3600,
			LOS:          true,
			Environment:  "urban",
		},
	}
	pathloss := GetPathLoss(model.Coordinate{Lat: 37.979207, Lng: 23.720989}, 1.5, cell)
	fmt.Printf("pathloss: %v", pathloss)
	// TxPowerDB := 40.0
	realizations := 1000

	//LOS
	PlotReceivedPower(pathloss, realizations, cell)

	//NLOS
	cell.Channel.LOS = false
	PlotReceivedPower(pathloss, realizations, cell)

}
