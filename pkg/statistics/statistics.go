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
