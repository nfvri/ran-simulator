package bandwidth

func MHzToHz(MHz float64) float64 {
	return MHz * 1e6
}

func HzToMHz(MHz float64) float64 {
	return MHz * 1e-6
}

func MHzToGHz(MHz float64) float64 {
	return MHz / 1e3
}
