package bandwidth

// arfcn -> nX
// nX -> FR1/FR2, Allowed CA combinations
// FR -> allowed SCS e.g. FR1 -> [15, 30, 60]
// SCS, ChannelBW -> Max #PRBs e.g. 15, 40MHz -> 216

const (
	UL = "Uplink"
	DL = "Downlink"
)

type Band struct {
	Name          string
	ULlow         float64 // Uplink low frequency
	ULhigh        float64 // Uplink high frequency
	DLlow         float64 // Downlink low frequency
	DLhigh        float64 // Downlink high frequency
	DuplexingMode string  // FDD/TDD
}

// bands contains the data for each NR operating band.
var bands = []Band{
	{"n1", 1920, 1980, 2110, 2170, "FDD"},
	{"n2", 1850, 1910, 1930, 1990, "FDD"},
	{"n3", 1710, 1785, 1805, 1880, "FDD"},
	{"n5", 824, 849, 869, 894, "FDD"},
	{"n7", 2500, 2570, 2620, 2690, "FDD"},
	{"n8", 880, 915, 925, 960, "FDD"},
	{"n12", 699, 716, 729, 746, "FDD"},
	{"n14", 788, 798, 758, 768, "FDD"},
	{"n18", 815, 830, 860, 875, "FDD"},
	{"n20", 832, 862, 791, 821, "FDD"},
	{"n25", 1850, 1915, 1930, 1995, "FDD"},
	{"n26", 814, 849, 859, 894, "FDD"},
	{"n28", 703, 748, 758, 803, "FDD"},
	{"n29", 0, 0, 717, 728, "SDL"},
	{"n30", 2305, 2315, 2350, 2360, "FDD"},
	{"n34", 2010, 2025, 2010, 2025, "TDD"},
	{"n38", 2570, 2620, 2570, 2620, "TDD"},
	{"n39", 1880, 1920, 1880, 1920, "TDD"},
	{"n40", 2300, 2400, 2300, 2400, "TDD"},
	{"n41", 2496, 2690, 2496, 2690, "TDD"},
	{"n48", 3550, 3700, 3550, 3700, "TDD"},
	{"n50", 1432, 1517, 1432, 1517, "TDD"},
	{"n51", 1427, 1432, 1427, 1432, "TDD"},
	{"n53", 2483.5, 2495, 2483.5, 2495, "TDD"},
	{"n65", 1920, 2010, 2110, 2200, "FDD"},
	{"n66", 1710, 1780, 2110, 2200, "FDD"},
	{"n70", 1695, 1710, 1995, 2020, "FDD"},
	{"n71", 663, 698, 617, 652, "FDD"},
	{"n74", 1427, 1470, 1475, 1518, "FDD"},
	{"n75", 0, 0, 1432, 1517, "SDL"},
	{"n76", 0, 0, 1427, 1432, "SDL"},
	{"n77", 3300, 4200, 3300, 4200, "TDD"},
	{"n78", 3300, 3800, 3300, 3800, "TDD"},
	{"n79", 4400, 5000, 4400, 5000, "TDD"},
	{"n80", 1710, 1785, 0, 0, "SUL"},
	{"n81", 880, 915, 0, 0, "SUL"},
	{"n82", 832, 862, 0, 0, "SUL"},
	{"n83", 703, 748, 0, 0, "SUL"},
	{"n84", 1920, 1980, 0, 0, "SUL"},
	{"n86", 1710, 1780, 0, 0, "SUL"},
	{"n89", 824, 849, 0, 0, "SUL"},
	{"n90", 2496, 2690, 2496, 2690, "TDD"},
	{"n91", 832, 862, 1427, 1432, "FDD2"},
	{"n92", 832, 862, 1432, 1517, "FDD2"},
	{"n93", 880, 915, 1427, 1432, "FDD2"},
	{"n94", 880, 915, 1432, 1517, "FDD2"},
	{"n95", 2010, 2025, 0, 0, "SUL"},
}

// GetBand takes a frequency and direction and returns the operating band name.
func GetBand(frequency float64, direction string) string {
	for _, band := range bands {
		if direction == UL && band.ULlow <= frequency && frequency <= band.ULhigh {
			return band.Name
		}
		if direction == DL && band.DLlow <= frequency && frequency <= band.DLhigh {
			return band.Name
		}
	}
	return "Unknown Band"
}

// GetFR takes a frequency and returns the frequency range designation.
func GetFR(frequency float64) string {
	switch {
	case frequency >= 410 && frequency <= 7125:
		return "FR1"
	case frequency >= 24250 && frequency <= 52600:
		return "FR2-1"
	case frequency > 52600 && frequency <= 71000:
		return "FR2-2"
	default:
		return "Out of Range"
	}
}
