package signal

import (
	"math"
	"math/rand"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"
)

// RicianRandom generates a random variable following the Rician distribution
// with parameters nu (non-centrality parameter) and sigma (scale parameter).
func RicianRandom(nu, sigma float64) float64 {
	// Generate two Gaussian random variables with mean 0 and standard deviation sigma
	x := (sigma * rand.NormFloat64()) + nu
	y := sigma * rand.NormFloat64()

	// Generate Rician-distributed random variable
	return math.Sqrt((x * x) + (y * y))
}

// Calculate nu and sigma from K-factor
func calculateNuSigma(K float64) (float64, float64) {
	// Assume total power P = 1
	sigma := math.Sqrt(1 / (2 * (K + 1)))
	nu := math.Sqrt(K / (K + 1))
	return nu, sigma
}

// RiceanFading calculates the channel fading using the rician fading model
func RiceanFading(K float64) float64 {
	nu, sigma := calculateNuSigma(K)
	// fadingAmplitude := complex(RicianRandom(nu, sigma), RicianRandom(0, sigma))
	fadingAmplitude := RicianRandom(nu, sigma)
	// TODO: define random sampling per environment & LOS/NLOS
	numSubPaths := 14.0
	for i := 0; i <= int(numSubPaths); i++ {
		fadingAmplitude += RicianRandom(nu, sigma)
	}

	return fadingAmplitude / numSubPaths
}

func GetRiceanK(cell *model.Cell) float64 {
	KdB := 9.0
	if cell.Channel.LOS {
		KdB = (rand.Float64() * RICEAN_K_STD_MACRO) + RICEAN_K_MEAN
	}
	K := utils.DbToDbm(KdB)
	return K
}
