package statistics

import "math"

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

// DRB.UEThpDl calculates the UE throughput in downlink.
func UEThpDl(totalDataDl int, duration float64) float64 {
	if totalDataDl == 0.0 && duration == 0.0 {
		return 0
	}
	if duration == 0.0 {
		return math.MaxFloat64
	}
	return float64(totalDataDl) / duration
}

// DRB.UEThpUl calculates the UE throughput in uplink.
func UEThpUl(totalDataUl int, duration float64) float64 {
	if totalDataUl == 0.0 && duration == 0.0 {
		return 0
	}
	if duration == 0.0 {
		return math.MaxFloat64
	}
	return float64(totalDataUl) / duration
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
