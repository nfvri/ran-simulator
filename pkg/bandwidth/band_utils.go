package bandwidth

const (
	UL    = "Uplink"
	DL    = "Downlink"
	FR1   = "FR1"
	FR2   = "FR2"
	FR2_1 = "FR2-1"
	FR2_2 = "FR2-2"
)

type BandEutra struct {
	Name          string
	ULlow         float64 // Uplink low frequency
	ULhigh        float64 // Uplink high frequency
	DLlow         float64 // Downlink low frequency
	DLhigh        float64 // Downlink high frequency
	DuplexingMode string  // FDD/TDD
}

var BandsEutra = map[string]BandEutra{
	"1":  {"1", 1920, 1980, 2110, 2170, "FDD"},
	"2":  {"2", 1850, 1910, 1930, 1990, "FDD"},
	"3":  {"3", 1710, 1785, 1805, 1880, "FDD"},
	"4":  {"4", 1710, 1755, 2110, 2155, "FDD"},
	"5":  {"5", 824, 849, 869, 894, "FDD"},
	"7":  {"7", 2500, 2570, 2620, 2690, "FDD"},
	"8":  {"8", 880, 915, 925, 960, "FDD"},
	"20": {"20", 832, 862, 791, 821, "FDD"},
	"28": {"28", 703, 748, 758, 803, "FDD"},
	"38": {"38", 2570, 2620, 2570, 2620, "TDD"},
	"40": {"40", 2300, 2400, 2300, 2400, "TDD"},
	"41": {"41", 2496, 2690, 2496, 2690, "TDD"},
	"66": {"66", 1710, 1780, 2110, 2200, "FDD"},
	"71": {"71", 663, 698, 617, 652, "FDD"},
}

// func FrequencyToEARFCN(frequency float64, isUplink bool) (int, error) {
// 	for _, band := range BandsEutra {
// 		if isUplink {
// 			if frequency >= band.ULlow && frequency <= band.ULhigh {
// 				return int((frequency - band.ULlow) * 10) + band.EarfcnULlow, nil
// 			}
// 		} else {
// 			if frequency >= band.DLlow && frequency <= band.DLhigh {
// 				return int((frequency - band.DLlow) * 10) + band.EarfcnDLlow, nil
// 			}
// 		}
// 	}
// 	return 0, fmt.Errorf("frequency out of range")
// }

type BandNR struct {
	Name              string
	ULlow             float64 // Uplink low frequency
	ULhigh            float64 // Uplink high frequency
	DLlow             float64 // Downlink low frequency
	DLhigh            float64 // Downlink high frequency
	DuplexingMode     string  // FDD/TDD
	ArfcnULlow        float64
	ArfcnULhigh       float64
	ArfcnDLlow        float64
	ArfcnDLhigh       float64
	ChannelBandwidths []int
}

var BandsNR = map[string]BandNR{

	"n1":   {"n1", 1920, 1980, 2110, 2170, "FDD", 384000, 396000, 422000, 434000, []int{}},
	"n2":   {"n2", 1850, 1910, 1930, 1990, "FDD", 370000, 382000, 386000, 398000, []int{}},
	"n3":   {"n3", 1710, 1785, 1805, 1880, "FDD", 342000, 357000, 361000, 376000, []int{}},
	"n5":   {"n5", 824, 849, 869, 894, "FDD", 164800, 169800, 173800, 178800, []int{}},
	"n7":   {"n7", 2500, 2570, 2620, 2690, "FDD", 500000, 514000, 524000, 538000, []int{}},
	"n8":   {"n8", 880, 915, 925, 960, "FDD", 176000, 183000, 185000, 192000, []int{}},
	"n12":  {"n12", 699, 716, 729, 746, "FDD", 139800, 143200, 145800, 149200, []int{}},
	"n13":  {"n13", 777, 787, 746, 756, "FDD", 155400, 157400, 149200, 151200, []int{}},
	"n14":  {"n14", 788, 798, 758, 768, "FDD", 157600, 159600, 151600, 153600, []int{}},
	"n18":  {"n18", 815, 830, 860, 875, "FDD", 163000, 166000, 172000, 175000, []int{}},
	"n20":  {"n20", 832, 862, 791, 821, "FDD", 166400, 172400, 158200, 164200, []int{}},
	"n24":  {"n24", 1626.5, 1660.5, 1525, 1559, "FDD", 325300, 332100, 305000, 311800, []int{}},
	"n25":  {"n25", 1850, 1915, 1930, 1995, "FDD", 370000, 383000, 386000, 399000, []int{}},
	"n26":  {"n26", 814, 849, 859, 894, "FDD", 162800, 169800, 171800, 178800, []int{}},
	"n28":  {"n28", 703, 748, 758, 803, "FDD", 140600, 149600, 151600, 160600, []int{}},
	"n29":  {"n29", 0, 0, 717, 728, "SDL", 0, 0, 143400, 145600, []int{}},
	"n30":  {"n30", 2305, 2315, 2350, 2360, "FDD", 461000, 463000, 470000, 472000, []int{}},
	"n31":  {"n31", 452.5, 457.5, 462.5, 467.5, "TDD", 90500, 91500, 92500, 93500, []int{}},
	"n34":  {"n34", 2010, 2025, 2010, 2025, "TDD", 402000, 405000, 402000, 405000, []int{}},
	"n38":  {"n38", 2570, 2620, 2570, 2620, "TDD", 514000, 524000, 514000, 524000, []int{}},
	"n39":  {"n39", 1880, 1920, 1880, 1920, "TDD", 376000, 384000, 376000, 384000, []int{}},
	"n40":  {"n40", 2300, 2400, 2300, 2400, "TDD", 460000, 480000, 460000, 480000, []int{}},
	"n41":  {"n41", 2496, 2690, 2496, 2690, "TDD", 499200, 537999, 499200, 537999, []int{}},
	"n46":  {"n46", 5150, 5925, 5150, 5925, "TDD", 743334, 795000, 743334, 795000, []int{}},
	"n47":  {"n47", 5855, 5925, 5855, 5925, "TDD", 790334, 795000, 790334, 795000, []int{}},
	"n48":  {"n48", 3550, 3700, 3550, 3700, "TDD", 636667, 646666, 636667, 646666, []int{}},
	"n50":  {"n50", 1432, 1517, 1432, 1517, "TDD", 286400, 303400, 286400, 303400, []int{}},
	"n51":  {"n51", 1427, 1432, 1427, 1432, "TDD", 285400, 286400, 285400, 286400, []int{}},
	"n53":  {"n53", 2483.5, 2495, 2483.5, 2495, "TDD", 496700, 499000, 496700, 499000, []int{}},
	"n54":  {"n54", 1670, 1675, 1670, 1675, "TDD", 334000, 335000, 334000, 335000, []int{}},
	"n65":  {"n65", 1920, 2010, 2110, 2200, "FDD", 384000, 402000, 422000, 440000, []int{}},
	"n66":  {"n66", 1710, 1780, 2110, 2200, "FDD", 342000, 356000, 422000, 440000, []int{}},
	"n67":  {"n67", 0, 0, 738, 758, "SDL", 0, 0, 147600, 151600, []int{}},
	"n70":  {"n70", 1695, 1710, 1995, 2020, "FDD", 339000, 342000, 399000, 404000, []int{}},
	"n71":  {"n71", 663, 698, 617, 652, "FDD", 132600, 139600, 123400, 130400, []int{}},
	"n72":  {"n72", 451, 456, 461, 466, "FDD", 90200, 91200, 92200, 93200, []int{}},
	"n74":  {"n74", 1427, 1470, 1475, 1518, "FDD", 285400, 294000, 295000, 303600, []int{}},
	"n75":  {"n75", 0, 0, 1432, 1517, "SDL", 0, 0, 286400, 303400, []int{}},
	"n76":  {"n76", 0, 0, 1427, 1432, "SDL", 0, 0, 285400, 286400, []int{}},
	"n77":  {"n77", 3300, 4200, 3300, 4200, "TDD", 620000, 680000, 620000, 680000, []int{}},
	"n78":  {"n78", 3300, 3800, 3300, 3800, "TDD", 620000, 653333, 620000, 653333, []int{}},
	"n79":  {"n79", 4400, 5000, 4400, 5000, "TDD", 693334, 733333, 693334, 733333, []int{}},
	"n80":  {"n80", 1710, 1785, 0, 0, "SUL", 342000, 357000, 0, 0, []int{}},
	"n81":  {"n81", 880, 915, 0, 0, "SUL", 176000, 183000, 0, 0, []int{}},
	"n82":  {"n82", 832, 862, 0, 0, "SUL", 166400, 172400, 0, 0, []int{}},
	"n83":  {"n83", 703, 748, 0, 0, "SUL", 140600, 149600, 0, 0, []int{}},
	"n84":  {"n84", 1920, 1980, 0, 0, "SUL", 384000, 396000, 0, 0, []int{}},
	"n85":  {"n85", 698, 716, 728, 746, "FDD", 139600, 143200, 145600, 149200, []int{}},
	"n86":  {"n86", 1710, 1780, 0, 0, "SUL", 342000, 356000, 0, 0, []int{}},
	"n89":  {"n89", 824, 849, 0, 0, "SUL", 164800, 169800, 0, 0, []int{}},
	"n90":  {"n90", 2496, 2690, 2496, 2690, "TDD", 499200, 537999, 499200, 537999, []int{}},
	"n91":  {"n91", 832, 862, 1427, 1432, "FDD", 166400, 172400, 285400, 286400, []int{}},
	"n92":  {"n92", 832, 862, 1432, 1517, "FDD", 166400, 172400, 286400, 303400, []int{}},
	"n93":  {"n93", 880, 915, 1427, 1432, "FDD", 176000, 183000, 285400, 286400, []int{}},
	"n94":  {"n94", 880, 915, 1432, 1517, "FDD", 176000, 183000, 286400, 303400, []int{}},
	"n95":  {"n95", 2010, 2025, 0, 0, "SUL", 402000, 405000, 0, 0, []int{}},
	"n96":  {"n96", 5925, 7125, 5925, 7125, "TDD", 795000, 875000, 795000, 875000, []int{}},
	"n97":  {"n97", 2300, 2400, 0, 0, "SUL", 460000, 480000, 0, 0, []int{}},
	"n98":  {"n98", 1880, 1920, 0, 0, "SUL", 376000, 384000, 0, 0, []int{}},
	"n99":  {"n99", 1626.5, 1660.5, 0, 0, "SUL", 325300, 332100, 0, 0, []int{}},
	"n100": {"n100", 874.4, 880, 919.4, 925, "FDD", 174880, 176000, 183880, 185000, []int{}},
	"n101": {"n101", 1900, 1910, 1900, 1910, "TDD", 380000, 382000, 380000, 382000, []int{}},
	"n102": {"n102", 5925, 6425, 5925, 6425, "TDD", 795000, 828333, 795000, 828333, []int{}},
	"n104": {"n104", 6425, 7125, 6425, 7125, "TDD", 828334, 875000, 828334, 875000, []int{}},
	"n105": {"n105", 663, 703, 612, 652, "FDD", 132600, 140600, 122400, 130400, []int{}},
	"n106": {"n106", 896, 901, 935, 940, "FDD", 179200, 180200, 187000, 188000, []int{}},
	"n109": {"n109", 703, 733, 1432, 1517, "FDD", 140600, 146600, 286400, 303400, []int{}},
	"n256": {"n256", 1980, 2010, 2170, 2200, "FDD", 396000, 402000, 434000, 440000, []int{}},
	"n255": {"n255", 1626.5, 1660.5, 1525, 1559, "FDD", 325300, 332100, 305000, 311800, []int{}},
	"n254": {"n254", 1610, 1626.5, 2483.5, 2500, "FDD", 322000, 325300, 496700, 500000, []int{}},
}

// type SCSInfo struct {
// 	Value      int    // Subcarrier Spacing
// 	Numerology int    // Numerology (μ)
// 	FRName     string // "FR1"/"FR2-1"/"FR2-2"
// }

// SCS mapping defines the SCS values, correspoding numerologies, anf frequency ranges.
var SupportedSCSByFR = map[string][]int{
	FR1: {15, 30, 60},        // FR1 SCSs
	FR2: {60, 120, 480, 960}, // FR1 SCSs
}

var NrSCSByCQIPerFR = map[string]map[int]int{
	FR1: {
		1:  60,
		2:  60,
		3:  60,
		4:  60,
		5:  60,
		6:  30,
		7:  30,
		8:  30,
		9:  30,
		10: 30,
		11: 15,
		12: 15,
		13: 15,
		14: 15,
		15: 15,
	},
	FR2: {
		1:  480,
		2:  480,
		3:  480,
		4:  480,
		5:  480,
		6:  120,
		7:  120,
		8:  120,
		9:  120,
		10: 120,
		11: 60,
		12: 60,
		13: 60,
		14: 60,
		15: 60,
	},
}

// NumerologyMapping defines the numerology value for each SCS.
// var NumerologyMapping = map[int]int{
// 	0: 15,
// 	1: 30,
// 	2: 60,
// 	3: 120,
// 	4: 240,
// 	5: 480,
// 	6: 960,
// }

// GetBand takes a frequency and direction and returns the operating band name.
func GetBand(arfcn uint32, direction string) (nrBand BandNR, found bool) {
	for b := range BandsNR {
		band := BandsNR[b]
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
	freq := calculateFrequency(int(arfcn))
	switch {
	case freq >= 410 && freq <= 7125:
		return "FR1"
	case freq >= 24250 && freq <= 71000:
		return "FR2"
	default:
		return "Out of Range"
	}
}

// CalculateNominalChannelSpacing computes the nominal channel spacing (F_spacing)
// based on the 3GPP TS 38.104 NR specification (Section 5.4.1.2).
func CalculateNominalChannelSpacing(BandwidthA, BandwidthB float64, channelRaster float64, deltaFRaster float64) []float64 {
	baseSpacing := (BandwidthA + BandwidthB) / 2
	var adjustments []float64

	switch channelRaster {
	case 100:
		adjustments = []float64{0}
	case 15:
		if deltaFRaster == 15 {
			adjustments = []float64{-5, 0, 5}
		} else if deltaFRaster == 30 {
			adjustments = []float64{-10, 0, 10}
		}
	case 60:
		if deltaFRaster == 60 {
			adjustments = []float64{-20, 0, 20}
		} else if deltaFRaster == 120 {
			adjustments = []float64{-40, 0, 40}
		}
	}

	var spacings []float64
	for _, adj := range adjustments {
		spacings = append(spacings, baseSpacing+adj)
	}

	return spacings
}

func CalculateARFCN(frequency float64) int {
	var deltaFGlobal float64
	var fRefOffs float64
	var nRefOffs int

	switch {
	case frequency < 3000:
		deltaFGlobal = 5
		fRefOffs = 0
		nRefOffs = 0
	case frequency < 24250:
		deltaFGlobal = 15
		fRefOffs = 3000
		nRefOffs = 600000
	default:
		deltaFGlobal = 60
		fRefOffs = 24250.08
		nRefOffs = 2016667
	}

	nRef := nRefOffs + int((frequency-fRefOffs)*1000/deltaFGlobal)
	return nRef
}

func calculateFrequency(arfcn int) float64 {
	var deltaFGlobal float64
	var fRefOffs float64
	var nRefOffs int

	switch {
	case arfcn < 600000:
		deltaFGlobal = 5
		fRefOffs = 0
		nRefOffs = 0
	case arfcn < 2016667:
		deltaFGlobal = 15
		fRefOffs = 3000
		nRefOffs = 600000
	default:
		deltaFGlobal = 60
		fRefOffs = 24250.08
		nRefOffs = 2016667
	}

	frequency := fRefOffs + float64(arfcn-nRefOffs)*deltaFGlobal/1000
	return frequency
}

// // GetSCS takes a frequency and returns the allowed SCSs.
// func GetSCS(frequencyRange string) []int {
// 	var allowedSCS []int

// 	// Iterate over the SCSList and add SCS values for the given frequency range
// 	for _, scs := ran {
// 		if scs.FRName == frequencyRange {
// 			allowedSCS = append(allowedSCS, scs.Value)
// 		}
// 	}

// 	// Return the slice of allowed SCS values for the given frequency range
// 	return allowedSCS
// }

// GetNumerology takes an SCS value and returns its corresponding numerology.
// func GetNumerology(scs int) (int, bool) {
// 	num, exists := NumerologyMapping[scs]
// 	return num, exists
// }

// ChannelSCSPRB defines the structure for Channel Bandwidth, SCS, and Max PRBs.
type ChannelSCSPRB struct {
	ChannelBWMHz uint32 // Channel Bandwidth in MHz
	SCS          int    // Subcarrier Spacing in kHz
	PRBs         int    // Maximum PRBs
}

// ChannelTable contains entries sorted by Channel Bandwidth for efficient selection.
var ChannelInfoByFR = map[string][]ChannelSCSPRB{
	FR1: {
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
	},
	FR2: {
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
	},
}

// GetPRBs returns the PRBs for the available Channel Bandwidth and SCS.
func GetPRBs(channelBWMHz uint32, scs int, fr string) (prbs int) {

	additionalPRBsFound := false
	for channelBWMHz > 0 {
		for _, entry := range ChannelInfoByFR[fr] {
			// Find the largest ChannelBW <= input BW and matching SCS
			additionalPRBsFound = channelBWMHz <= entry.ChannelBWMHz && entry.SCS == scs
			if additionalPRBsFound {
				prbs += entry.PRBs
				channelBWMHz -= 12 * uint32(entry.PRBs) * uint32(entry.SCS)
			}
		}

		if !additionalPRBsFound {
			break
		}
	}

	return prbs
}
