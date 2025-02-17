package statistics

import (
	"fmt"

	bw "github.com/nfvri/ran-simulator/pkg/bandwidth"
)

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

// CalculateThroughputMbpsNR_CA calculates the maximum data rate based on 3GPP TS 38.306
//
// J - number of CCs aggregated, MAX_VALUE = 16 (3GPP 38.802)
//
// v(j)Layers - maximum number of MIMO layers , 3GPP 38.802: maximum 8 in DL, maximum 4 in UL
//
// MCS index -> Q(j)m, Rmax
//
// Q(j)m modulation order (3GPP 38.804, 38.214) is the maximum supported modulation order given by higher layer parameter
// supportedModulationOrderDL for downlink and higher layer parameter supportedModulationOrderUL for uplink
// For UL and DL Q(j)m is same (QPSK-2, 16QAM-4, 64QAM-6, 256QAM-8, 1024QAM-10)
//
// f(j) Scaling factor (3GPP 38.306) scaling factor given by higher layer parameter scalingFactor or
// scalingFactor-1024QAM-FR1 and can take the values 1, 0.8, 0.75, and 0.4
//
// µ(j) - numerology of carrier (3GPP 38.211)
//
// # Tµ,s is the average OFDM symbol duration in a subframe for numerology μ
//
// NbwPRB(j),μ -maximum number of PRB (3GPP 38.104) for selected BW(j), FR(j), µ(j).
func CalculateThroughputMbpsNR_CA(J int, N []int, Qm []int, v []int, f []float64, SCS []float64, fr, direction string) float64 {
	Rmax := 948.0 / 1024.0
	throughput := 0.0

	for j := 0; j < J; j++ {
		Ts := 1.0 / (SCS[j] * 1000.0) // Symbol duration in seconds
		Tslot := 14.0 * Ts            // Slot duration
		OH := bw.OH[bw.OHKey{FR: fr, Direction: direction}]
		throughput += float64(v[j]) * float64(Qm[j]) * f[j] * Rmax * float64(N[j]) * 12.0 / Tslot * (1 - OH)
	}

	return 1e-6 * throughput
}

// DataRateMbpsEUTRA_MRDC calculates the approximate maximum data rate for EUTRA in MR-DC (in Mbps).
// J: Number of aggregated EUTRA component carriers in MR-DC band combination
// tbs: Slice containing the total maximum number of DL-SCH or UL-SCH transport block bits for each component carrier.
func DataRateMbpsEUTRA_MRDC(J int, tbs []int) float64 {
	if len(tbs) != J {
		fmt.Println("Error: The length of the TBS slice must match the number of component carriers (J).")
		return 0.0
	}

	var sumTBS int
	for _, t := range tbs {
		sumTBS += t
	}

	dataRate := 1e-3 * float64(sumTBS)
	return dataRate
}
