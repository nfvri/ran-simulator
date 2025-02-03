package statistics

// DRB.MeanActiveUeDl calculates the mean number of active UEs in downlink.
func MeanActiveUeDl(activeUeCounts []int) float64 {
	sum := 0
	for _, count := range activeUeCounts {
		sum += count
	}
	return float64(sum) / float64(len(activeUeCounts))
}

// DRB.MeanActiveUeDl.QOS calculates the mean number of active UEs in downlink with a certain QoS class.
func MeanActiveUeDlQOS(activeUeCounts map[int]int, cqi int) float64 {
	sum := 0
	count := 0
	for index, value := range activeUeCounts {
		if index == cqi {
			sum += value
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return float64(sum) / float64(count)
}

// DRB.UEThp calculates the UE throughput in downlink.
func UEThp(usedPRBs int, numUEs int) float64 {
	if usedPRBs == 0 && numUEs == 0 {
		return 0
	}
	if numUEs == 0 {
		return -1
	}
	return float64(usedPRBs) / float64(numUEs)
}

// DRB.UEThpDl.QOS calculates the UE throughput in downlink with a certain QoS class.
func UEThpDlQOS(totalDataDlQOS map[int]int, duration float64, cqi int) float64 {
	data, exists := totalDataDlQOS[cqi]
	if !exists {
		return 0
	}
	return float64(data) / duration
}

// RRU.PrbUsedDl.QOS calculates the PRBs used in downlink with a certain QoS class.
func PrbUsedDlQOS(prbs map[int]int, cqi int) int {
	prb, exists := prbs[cqi]
	if !exists {
		return 0
	}
	return prb
}

// RRU.PrbUsedUl.QOS calculates the PRBs used in uplink with a certain QoS class.
func PrbUsedUlQOS(prbs map[int]int, cqi int) int {
	prb, exists := prbs[cqi]
	if !exists {
		return 0
	}
	return prb
}

// CalculateThroughputMbps calculates the maximum data rate based on 3GPP TS 38.306
func CalculateThroughputMbps(J int, N []int, Qm []int, v []int, f []float64, SCS []float64) float64 {
	Rmax := 948.0 / 1024.0
	throughput := 0.0

	for j := 0; j < J; j++ {
		Ts := 1.0 / (SCS[j] * 1000.0) // Symbol duration in seconds
		Tslot := 14.0 * Ts            // Slot duration
		throughput += float64(N[j]) * 12.0 * Rmax * float64(Qm[j]) * float64(v[j]) * f[j] / Tslot
	}

	return throughput * 1e-6 // Convert to Mbps
}
