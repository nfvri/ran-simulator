package signal

import (
	"fmt"
	"path/filepath"
	"testing"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// PlotReceivedPower plots the received power values and saves it as a PNG file
func PlotReceivedPower(pathlossDb float64, K float64, realizations int) {
	receivedPowerDb := make(plotter.XYs, realizations)

	for i := 0; i < realizations; i++ {
		receivedPowerDb[i].X = float64(i)
		receivedPowerDb[i].Y = MultipathFading(pathlossDb, false)
	}

	p := plot.New()
	p.Title.Text = fmt.Sprintf("Received Power (dB) - K=%.2f", K)
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

	receivedPowerFilename := filepath.Join("./multipath_test_results/graphs/", fmt.Sprintf("%s_K%.2f.png", "multipath_db", K))
	if err := p.Save(15*vg.Inch, 10*vg.Inch, receivedPowerFilename); err != nil {
		panic(err)
	}

	fmt.Printf("Plot saved to %s\n", receivedPowerFilename)

	// if err := os.MkdirAll("/multipath_test_results/distributions/", os.ModePerm); err != nil {
	// 	panic(err)
	// }

	// Plot the distribution separately
	distributionFilename := filepath.Join("./multipath_test_results/distributions/", fmt.Sprintf("%s_distribution_K%.2f.png", "multipath_db_", K))

	distribution := make(plotter.Values, len(receivedPowerDb))
	for i, pt := range receivedPowerDb {
		distribution[i] = pt.Y
	}

	pl := plot.New()
	pl.Title.Text = "Received Power Distribution"
	pl.X.Label.Text = "Received Power (dB)"
	pl.Y.Label.Text = "Frequency"

	h, err := plotter.NewHist(distribution, 16) // 16 bins for the histogram
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
	pathloss := 80.0
	// TxPowerDB := 40.0
	realizations := 500
	K_start := 4.0
	K_increment := 1.0
	num_tests := 9

	for i := 0; i < num_tests; i++ {
		K := K_start + float64(i)*K_increment
		PlotReceivedPower(pathloss, K, realizations)
	}
}
