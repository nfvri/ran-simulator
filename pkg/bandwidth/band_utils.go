package bandwidth

import "errors"

// arfcn -> nX
// nX -> FR1/FR2, Allowed CA combinations
// FR -> allowed SCS e.g. FR1 -> [15, 30, 60]
// SCS, ChannelBW -> Max #PRBs e.g. 15, 40MHz -> 216

const (
	UL = "Uplink"
	DL = "Downlink"
)

// TODO: determine if needed/how it will be used
// NRBandwidthClass represents a single row of the table.
type NRBandwidthClass struct {
	Class             string `json:"class"`
	AggregateBWMinMHz int    `json:"aggregate_bw_min_mhz"`
	AggregateBWMaxMHz string `json:"aggregate_bw_max_mhz"`
	NumContiguousCC   int    `json:"num_contiguous_cc"`
	FallbackGroup     []int  `json:"fallback_group"`
}

// TODO: determine if needed/how it will be used
var classes = []NRBandwidthClass{
	{"A", 0, "1 x BWChannel,max", 1, []int{1, 2, 3}},
	{"B", 20, "100", 2, []int{2, 3}},
	{"C", 100, "2 x BWChannel,max", 2, []int{1, 3}},
	{"D", 200, "3 x BWChannel,max", 3, []int{1, 3}},
	{"E", 300, "4 x BWChannel,max", 4, []int{1, 3}},
	{"G", 100, "150", 3, []int{2}},
	{"H", 150, "200", 4, []int{2}},
	{"I", 200, "250", 5, []int{2}},
	{"J", 250, "300", 6, []int{2}},
	{"K", 300, "350", 7, []int{2}},
	{"L", 350, "400", 8, []int{2}},
	{"M", 50, "200", 3, []int{3}},
	{"N", 80, "300", 4, []int{3}},
	{"O", 100, "400", 5, []int{3}},
}

type BandNR struct {
	Name          string
	ULlow         float64 // Uplink low frequency
	ULhigh        float64 // Uplink high frequency
	DLlow         float64 // Downlink low frequency
	DLhigh        float64 // Downlink high frequency
	DuplexingMode string  // FDD/TDD
}

// nrBands contains the data for each NR operating band.
var nrBands = map[string]BandNR{
	"n1":  {"n1", 1920, 1980, 2110, 2170, "FDD"},
	"n2":  {"n2", 1850, 1910, 1930, 1990, "FDD"},
	"n3":  {"n3", 1710, 1785, 1805, 1880, "FDD"},
	"n5":  {"n5", 824, 849, 869, 894, "FDD"},
	"n7":  {"n7", 2500, 2570, 2620, 2690, "FDD"},
	"n8":  {"n8", 880, 915, 925, 960, "FDD"},
	"n12": {"n12", 699, 716, 729, 746, "FDD"},
	"n14": {"n14", 788, 798, 758, 768, "FDD"},
	"n18": {"n18", 815, 830, 860, 875, "FDD"},
	"n20": {"n20", 832, 862, 791, 821, "FDD"},
	"n25": {"n25", 1850, 1915, 1930, 1995, "FDD"},
	"n26": {"n26", 814, 849, 859, 894, "FDD"},
	"n28": {"n28", 703, 748, 758, 803, "FDD"},
	"n29": {"n29", 0, 0, 717, 728, "SDL"},
	"n30": {"n30", 2305, 2315, 2350, 2360, "FDD"},
	"n34": {"n34", 2010, 2025, 2010, 2025, "TDD"},
	"n38": {"n38", 2570, 2620, 2570, 2620, "TDD"},
	"n39": {"n39", 1880, 1920, 1880, 1920, "TDD"},
	"n40": {"n40", 2300, 2400, 2300, 2400, "TDD"},
	"n41": {"n41", 2496, 2690, 2496, 2690, "TDD"},
	"n48": {"n48", 3550, 3700, 3550, 3700, "TDD"},
	"n50": {"n50", 1432, 1517, 1432, 1517, "TDD"},
	"n51": {"n51", 1427, 1432, 1427, 1432, "TDD"},
	"n53": {"n53", 2483.5, 2495, 2483.5, 2495, "TDD"},
	"n65": {"n65", 1920, 2010, 2110, 2200, "FDD"},
	"n66": {"n66", 1710, 1780, 2110, 2200, "FDD"},
	"n70": {"n70", 1695, 1710, 1995, 2020, "FDD"},
	"n71": {"n71", 663, 698, 617, 652, "FDD"},
	"n74": {"n74", 1427, 1470, 1475, 1518, "FDD"},
	"n75": {"n75", 0, 0, 1432, 1517, "SDL"},
	"n76": {"n76", 0, 0, 1427, 1432, "SDL"},
	"n77": {"n77", 3300, 4200, 3300, 4200, "TDD"},
	"n78": {"n78", 3300, 3800, 3300, 3800, "TDD"},
	"n79": {"n79", 4400, 5000, 4400, 5000, "TDD"},
	"n80": {"n80", 1710, 1785, 0, 0, "SUL"},
	"n81": {"n81", 880, 915, 0, 0, "SUL"},
	"n82": {"n82", 832, 862, 0, 0, "SUL"},
	"n83": {"n83", 703, 748, 0, 0, "SUL"},
	"n84": {"n84", 1920, 1980, 0, 0, "SUL"},
	"n86": {"n86", 1710, 1780, 0, 0, "SUL"},
	"n89": {"n89", 824, 849, 0, 0, "SUL"},
	"n90": {"n90", 2496, 2690, 2496, 2690, "TDD"},
	"n91": {"n91", 832, 862, 1427, 1432, "FDD2"},
	"n92": {"n92", 832, 862, 1432, 1517, "FDD2"},
	"n93": {"n93", 880, 915, 1427, 1432, "FDD2"},
	"n94": {"n94", 880, 915, 1432, 1517, "FDD2"},
	"n95": {"n95", 2010, 2025, 0, 0, "SUL"},
}

type SCSInfo struct {
	Value      int    // Subcarrier Spacing
	Numerology int    // Numerlogy (μ)
	FRName     string // "FR1"/"FR2-1"/"FR2-2"
}

// SCS mapping defines the SCS values, correspoding numerologies, anf frequency ranges.
var SCSList = []SCSInfo{
	{15, 0, "FR1"},    // SCS 15 kHz is allowed in FR1
	{30, 1, "FR1"},    // SCS 30 kHz is allowed in FR1
	{60, 2, "FR2-1"},  // SCS 60 kHz spans FR1 and FR2
	{120, 3, "FR2-2"}, // SCS 120 kHz is allowed in FR2
	{240, 4, "FR2-2"}, // SCS 240 kHz is allowed in FR2
	{480, 5, "FR2-2"}, // SCS 4800 kHz is allowed in FR2
	{960, 6, "FR2-2"}, // SCS 4800 kHz is allowed in FR2
}

// NumerologyMapping defines the numerology value for each SCS.
var NumerologyMapping = map[int]int{
	0: 15,
	1: 30,
	2: 60,
	3: 120,
	4: 240,
	5: 480,
	6: 960,
}

// GetBand takes a frequency and direction and returns the operating band name.
func GetBand(arfcn uint32, direction string) (nrBand BandNR, found bool) {
	for b := range nrBands {
		band := nrBands[b]
		arfcnFloat := float64(arfcn)

		found = direction == UL && band.ULlow <= arfcnFloat && arfcnFloat <= band.ULhigh
		if found {
			return band, found
		}

		found = direction == DL && band.DLlow <= arfcnFloat && arfcnFloat <= band.DLhigh
		if found {
			return band, found
		}
	}
	return
}

// GetFR takes a frequency and returns the frequency range designation.
func GetFR(arfcn float64) string {
	switch {
	case arfcn >= 410 && arfcn <= 7125:
		return "FR1"
	case arfcn >= 24250 && arfcn <= 52600:
		return "FR2-1"
	case arfcn > 52600 && arfcn <= 71000:
		return "FR2-2"
	default:
		return "Out of Range"
	}
}

// GetSCS takes a frequency and returns the allowed SCSs.
func GetSCS(frequencyRange string) []int {
	var allowedSCS []int

	// Iterate over the SCSList and add SCS values for the given frequency range
	for _, scs := range SCSList {
		if scs.FRName == frequencyRange {
			allowedSCS = append(allowedSCS, scs.Value)
		}
	}

	// Return the slice of allowed SCS values for the given frequency range
	return allowedSCS
}

// GetNumerology takes an SCS value and returns its corresponding numerology.
func GetNumerology(scs int) (int, bool) {
	num, exists := NumerologyMapping[scs]
	return num, exists
}

// ChannelSCSPRB defines the structure for Channel Bandwidth, SCS, and Max PRBs.
type ChannelSCSPRB struct {
	ChannelBW float64 // Channel Bandwidth in MHz
	SCS       int     // Subcarrier Spacing in kHz
	PRBs      int     // Maximum PRBs
}

// ChannelTable contains entries sorted by Channel Bandwidth for efficient selection.
var ChannelTableFR1 = []ChannelSCSPRB{
	{5, 15, 25},    // 5 MHz, SCS 15 kHz, 25 PRBs
	{5, 30, 11},    //        SCS 15 kHz, 11 PRBs
	{10, 15, 52},   // 10 MHz, SCS 15 kHz, 52 PRBs
	{10, 30, 24},   //         SCS 30 kHz, 24 PRBs
	{10, 60, 11},   //         SCS 60 kHz, 11 PRBs
	{15, 15, 79},   // 15 MHz, SCS 15 kHz, 79 PRBs
	{15, 30, 38},   //         SCS 30 kHz, 38 PRBs
	{15, 60, 18},   //         SCS 60 kHz, 18 PRBs
	{20, 15, 106},  // 20 MHz, SCS 15 kHz, 106 PRBs
	{20, 30, 51},   //         SCS 30 kHz, 51 PRBs
	{20, 60, 24},   //         SCS 30 kHz, 24 PRBs
	{25, 15, 106},  // 25 MHz, SCS 15 kHz, 106 PRBs
	{25, 30, 51},   //         SCS 30 kHz, 51 PRBs
	{25, 60, 24},   //         SCS 30 kHz, 24 PRBs
	{30, 15, 160},  // 30 MHz, SCS 15 kHz, 160 PRBs
	{30, 30, 78},   //         SCS 30 kHz, 78 PRBs
	{30, 60, 38},   //         SCS 30 kHz, 38 PRBs
	{35, 15, 188},  // 35 MHz, SCS 15 kHz, 188 PRBs
	{35, 30, 92},   //         SCS 30 kHz, 92 PRBs
	{35, 60, 44},   //         SCS 30 kHz, 44 PRBs
	{40, 15, 216},  // 40 MHz, SCS 15 kHz, 216 PRBs
	{40, 30, 106},  //         SCS 30 kHz, 106 PRBs
	{40, 60, 51},   //         SCS 60 kHz, 51 PRBs
	{45, 15, 242},  // 45 MHz, SCS 15 kHz, 242 PRBs
	{45, 30, 119},  //         SCS 30 kHz, 119 PRBs
	{45, 60, 58},   //         SCS 60 kHz, 58 PRBs
	{50, 15, 270},  // 50 MHz, SCS 15 kHz, 270 PRBs
	{50, 30, 133},  //         SCS 30 kHz, 133 PRBs
	{50, 60, 65},   //         SCS 30 kHz, 65 PRBs
	{60, 30, 162},  // 60 MHz, SCS 30 kHz, 162 PRBs
	{60, 60, 79},   //         SCS 60 kHz, 79 PRBs
	{70, 30, 189},  // 70 MHz, SCS 30 kHz, 189 PRBs
	{70, 60, 93},   //         SCS 60 kHz, 93 PRBs
	{80, 30, 217},  // 80 MHz, SCS 30 kHz, 217 PRBs
	{80, 60, 107},  //         SCS 60 kHz,107 PRBs
	{90, 30, 245},  // 90 MHz, SCS 30 kHz, 245 PRBs
	{90, 60, 121},  //         SCS 60 kHz, 121 PRBs
	{100, 30, 273}, // 100 MHz, SCS 30 kHz, 273 PRBs
	{100, 60, 136}, //          SCS 60 kHz, 136 PRBs
}

var ChannelTableFR2 = []ChannelSCSPRB{
	{50, 60, 66},
	{50, 120, 32},
	{100, 60, 132},
	{100, 120, 66},
	{200, 60, 264},
	{200, 120, 132},
	{400, 120, 264},
	{400, 480, 66},
	{400, 960, 33},
	{800, 480, 124},
	{800, 960, 61},
	{1600, 480, 248},
	{1600, 960, 124},
	{2000, 960, 148},
}

// GetPRBs returns the PRBs for the available Channel Bandwidth and SCS.
func GetPRBs(channelBW float64, scs int, isFR2 bool) (int, error) {
	var selectedEntry *ChannelSCSPRB
	var table []ChannelSCSPRB

	// Select the appropriate table based on FR1 or FR2
	if isFR2 {
		table = ChannelTableFR2 // Use FR2 table
	} else {
		table = ChannelTableFR1 // Use FR1 table
	}

	for _, entry := range table {
		// Find the largest ChannelBW <= input BW and matching SCS
		if entry.ChannelBW <= channelBW && entry.SCS == scs {
			if selectedEntry == nil || entry.ChannelBW > selectedEntry.ChannelBW {
				selectedEntry = &entry
			}
		}
	}

	// Return an error if no matching entry is found
	if selectedEntry == nil {
		return 0, errors.New("Not Applicable combination of Channel Bandwidth and SCS")
	}

	return selectedEntry.PRBs, nil
}
