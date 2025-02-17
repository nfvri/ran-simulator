package bandwidth

import (
	"math"
)

const (
	UL    = "Uplink"
	DL    = "Downlink"
	FR1   = "FR1"
	FR2   = "FR2"
	FR2_1 = "FR2-1"
	FR2_2 = "FR2-2"
)

// Duplex modes
const (
	FDD = "FDD"
	TDD = "TDD"
	SUL = "SUL"
	SDL = "SDL"
)

type BandEutra struct {
	Name          string
	ULlow         float64 // Uplink low frequency
	ULhigh        float64 // Uplink high frequency
	DLlow         float64 // Downlink low frequency
	DLhigh        float64 // Downlink high frequency
	EarfcnULlow   float64
	EarfcnULhigh  float64
	EarfcnDLlow   float64
	EarfcnDLhigh  float64
	DuplexingMode string // FDD/TDD/SUL/SDL
}

var BandsEutra = map[string]BandEutra{
	"1":  {"1", 1920, 1980, 2110, 2170, 19200, 19949, 0, 599, FDD},
	"2":  {"2", 1850, 1910, 1930, 1990, 18600, 19199, 600, 1199, FDD},
	"3":  {"3", 1710, 1785, 1805, 1880, 19200, 19949, 1200, 1949, FDD},
	"4":  {"4", 1710, 1755, 2110, 2155, 17100, 17549, 1950, 2399, FDD},
	"5":  {"5", 824, 849, 869, 894, 20400, 20649, 2400, 2649, FDD},
	"7":  {"7", 2500, 2570, 2620, 2690, 20700, 21449, 2750, 3449, FDD},
	"8":  {"8", 880, 915, 925, 960, 21450, 21799, 3450, 3799, FDD},
	"20": {"20", 832, 862, 791, 821, 22100, 22349, 6150, 6449, FDD},
	"28": {"28", 703, 748, 758, 803, 27000, 27449, 9210, 9659, FDD},
	"38": {"38", 2570, 2620, 2570, 2620, 25700, 26199, 25700, 26199, TDD},
	"40": {"40", 2300, 2400, 2300, 2400, 23000, 23999, 23000, 23999, TDD},
	"41": {"41", 2496, 2690, 2496, 2690, 24960, 26899, 24960, 26899, TDD},
	"66": {"66", 1710, 1780, 2110, 2200, 17100, 17849, 66436, 67335, FDD},
	"71": {"71", 663, 698, 617, 652, 68586, 68935, 68586, 68935, FDD},
}

type BandNR struct {
	Name                   string
	ULlow                  float64 // Uplink low frequency
	ULhigh                 float64 // Uplink high frequency
	DLlow                  float64 // Downlink low frequency
	DLhigh                 float64 // Downlink high frequency
	DuplexingMode          string  // FDD/TDD/SUL/SDL
	ArfcnULlow             float64
	ArfcnULhigh            float64
	ArfcnDLlow             float64
	ArfcnDLhigh            float64
	ChannelBandwidthsBySCS map[int][]int
}

var ChannelBWByFR = map[string][]int{
	FR1: {3, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 60, 70, 80, 90, 100},
	FR2: {50, 100, 200, 400},
}
var BandsNR = map[string]BandNR{
	// FR1
	"n1": {
		"n1",
		1920,
		1980,
		2110,
		2170,
		FDD,
		384000,
		396000,
		422000,
		434000,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 40, 45, 50},
			30: {10, 15, 20, 25, 30, 40, 45, 50},
			60: {10, 15, 20, 25, 30, 40, 45, 50},
		},
	},
	"n2": {
		"n2",
		1850,
		1910,
		1930,
		1990,
		FDD,
		370000,
		382000,
		386000,
		398000,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 35, 40},
			30: {10, 15, 20, 25, 30, 35, 40},
			60: {10, 15, 20, 25, 30, 35, 40},
		},
	},

	"n3": {
		"n3",
		1710,
		1785,
		1805,
		1880,
		FDD,
		342000,
		357000,
		361000,
		376000,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 35, 40, 45, 50},
			30: {10, 15, 20, 25, 30, 35, 40, 45, 50},
			60: {10, 15, 20, 25, 30, 35, 40, 45, 50},
		},
	},
	"n5": {
		"n5",
		824,
		849,
		869,
		894,
		FDD,
		164800,
		169800,
		173800,
		178800,
		map[int][]int{
			15: {5, 10, 15, 20, 25},
			30: {10, 15, 20, 25},
			60: {},
		},
	},
	"n7": {
		"n7",
		2500,
		2570,
		2620,
		2690,
		FDD,
		500000,
		514000,
		524000,
		538000,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 35, 40, 50},
			30: {10, 15, 20, 25, 30, 35, 40, 50},
			60: {10, 15, 20, 25, 30, 35, 40, 50},
		},
	},
	"n8": {
		"n8",
		880,
		915,
		925,
		960,
		FDD,
		176000,
		183000,
		185000,
		192000,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 35},
			30: {10, 15, 20, 25, 30, 35},
			60: {},
		},
	},
	"n12": {
		"n12",
		699,
		716,
		729,
		746,
		FDD,
		139800,
		143200,
		145800,
		149200,
		map[int][]int{
			15: {5, 10, 15},
			30: {10, 15},
			60: {},
		},
	},
	"n13": {
		"n13",
		777,
		787,
		746,
		756,
		FDD,
		155400,
		157400,
		149200,
		151200,
		map[int][]int{
			15: {5, 10},
			30: {10},
			60: {},
		},
	},
	"n14": {
		"n14",
		788,
		798,
		758,
		768,
		FDD,
		157600,
		159600,
		151600,
		153600,
		map[int][]int{
			15: {5, 10},
			30: {10},
			60: {},
		},
	},
	"n18": {
		"n18",
		815,
		830,
		860,
		875,
		FDD,
		163000,
		166000,
		172000,
		175000,
		map[int][]int{
			15: {5, 10, 15},
			30: {10, 15},
			60: {},
		},
	},
	"n20": {
		"n20",
		832,
		862,
		791,
		821,
		FDD,
		166400,
		172400,
		158200,
		164200,
		map[int][]int{
			15: {5, 10, 15, 20},
			30: {10, 15, 20},
			60: {},
		},
	},
	"n24": {
		"n24",
		1626.5,
		1660.5,
		1525,
		1559,
		FDD,
		325300,
		332100,
		305000,
		311800,
		map[int][]int{
			15: {5, 10},
			30: {10},
			60: {10},
		},
	},
	"n25": {
		"n25",
		1850,
		1915,
		1930,
		1995,
		FDD,
		370000,
		383000,
		386000,
		399000,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 35, 40, 45},
			30: {10, 15, 20, 25, 30, 35, 40, 45},
			60: {10, 15, 20, 25, 30, 35, 40, 45},
		},
	},
	"n26": {
		"n26",
		814,
		849,
		859,
		894,
		FDD,
		162800,
		169800,
		171800,
		178800,
		map[int][]int{
			15: {3, 5, 10, 15, 20, 25, 30},
			30: {10, 15, 20, 25, 30},
			60: {},
		},
	},
	"n28": {
		"n28",
		703,
		748,
		758,
		803,
		FDD,
		140600,
		149600,
		151600,
		160600,
		map[int][]int{
			15: {3, 5, 10, 15, 20, 25, 30, 40},
			30: {10, 15, 20, 25, 30, 40},
			60: {},
		},
	},
	"n29": {
		"n29",
		0,
		0,
		717,
		728,
		SDL,
		0,
		0,
		143400,
		145600,
		map[int][]int{
			15: {5, 10},
			30: {10},
			60: {},
		},
	},
	"n30": {
		"n30",
		2305,
		2315,
		2350,
		2360,
		FDD,
		461000,
		463000,
		470000,
		472000,
		map[int][]int{
			15: {5, 10},
			30: {10},
			60: {},
		},
	},
	"n31": {
		"n31",
		452.5,
		457.5,
		462.5,
		467.5,
		TDD,
		90500,
		91500,
		92500,
		93500,
		map[int][]int{
			15: {3, 5},
			30: {},
			60: {},
		},
	},
	"n34": {
		"n34",
		2010,
		2025,
		2010,
		2025,
		TDD,
		402000,
		405000,
		402000,
		405000,
		map[int][]int{
			15: {5, 10, 15},
			30: {10, 15},
			60: {10, 15},
		},
	},
	"n38": {
		"n38",
		2570,
		2620,
		2570,
		2620,
		TDD,
		514000,
		524000,
		514000,
		524000,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 40},
			30: {10, 15, 20, 25, 30, 40},
			60: {10, 15, 20, 25, 30, 40},
		},
	},
	"n39": {
		"n39",
		1880,
		1920,
		1880,
		1920,
		TDD,
		376000,
		384000,
		376000,
		384000,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 35, 40},
			30: {10, 15, 20, 25, 30, 35, 40},
			60: {10, 15, 20, 25, 30, 35, 40},
		},
	},
	"n40": {
		"n40",
		2300,
		2400,
		2300,
		2400,
		TDD,
		460000,
		480000,
		460000,
		480000,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 40, 50},
			30: {10, 15, 20, 25, 30, 40, 50, 60, 70, 80, 90, 100},
			60: {10, 15, 20, 25, 30, 40, 50, 60, 70, 80, 90, 100},
		},
	},
	"n41": {
		"n41",
		2496,
		2690,
		2496,
		2690,
		TDD,
		499200,
		537999,
		499200,
		537999,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 35, 40, 45, 50},
			30: {10, 15, 20, 25, 30, 35, 40, 45, 50, 60, 70, 80, 90, 100},
			60: {10, 15, 20, 25, 30, 35, 40, 45, 50, 60, 70, 80, 90, 100},
		},
	},
	"n46": {
		"n46",
		5150,
		5925,
		5150,
		5925,
		TDD,
		743334,
		795000,
		743334,
		795000,
		map[int][]int{
			15: {10, 20, 40},
			30: {10, 20, 40, 60, 80, 100},
			60: {10, 20, 40, 60, 80, 100},
		},
	},
	"n47": {
		"n47",
		5855,
		5925,
		5855,
		5925,
		TDD,
		790334,
		795000,
		790334,
		795000,
		map[int][]int{
			15: {10, 20, 30, 40},
			30: {10, 20, 30, 40},
			60: {10, 20, 30, 40},
		},
	},
	"n48": {
		"n48",
		3550,
		3700,
		3550,
		3700,
		TDD,
		636667,
		646666,
		636667,
		646666,
		map[int][]int{
			15: {5, 10, 15, 20, 30, 40, 50},
			30: {10, 15, 20, 30, 40, 50, 60, 70, 80, 90, 100},
			60: {10, 15, 20, 30, 40, 50, 60, 70, 80, 90, 100},
		},
	},
	"n50": {
		"n50",
		1432,
		1517,
		1432,
		1517,
		TDD,
		286400,
		303400,
		286400,
		303400,
		map[int][]int{
			15: {5, 10, 15, 20, 30, 40, 50},
			30: {10, 15, 20, 30, 40, 50, 60, 80},
			60: {10, 15, 20, 30, 40, 50, 60, 80},
		},
	},
	"n51": {
		"n51",
		1427,
		1432,
		1427,
		1432,
		TDD,
		285400,
		286400,
		285400,
		286400,
		map[int][]int{
			15: {5},
			30: {},
			60: {},
		},
	},
	"n53": {
		"n53",
		2483.5,
		2495,
		2483.5,
		2495,
		TDD,
		496700,
		499000,
		496700,
		499000,
		map[int][]int{
			15: {5, 10},
			30: {10},
			60: {10},
		},
	},
	"n54": {
		"n54",
		1670,
		1675,
		1670,
		1675,
		TDD,
		334000,
		335000,
		334000,
		335000,
		map[int][]int{
			15: {5},
			30: {},
			60: {},
		},
	},
	"n65": {
		"n65",
		1920,
		2010,
		2110,
		2200,
		FDD,
		384000,
		402000,
		422000,
		440000,
		map[int][]int{
			15: {5, 10, 15, 20, 50},
			30: {10, 15, 20, 50},
			60: {10, 15, 20, 50},
		},
	},
	"n66": {
		"n66",
		1710,
		1780,
		2110,
		2200,
		FDD,
		342000,
		356000,
		422000,
		440000,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 35, 40, 45},
			30: {10, 15, 20, 25, 30, 35, 40, 45},
			60: {10, 15, 20, 25, 30, 35, 40, 45},
		},
	},
	"n67": {
		"n67",
		0,
		0,
		738,
		758,
		SDL,
		0,
		0,
		147600,
		151600,
		map[int][]int{
			15: {5, 10, 15, 20},
			30: {10, 15, 20},
			60: {},
		},
	},
	"n70": {
		"n70",
		1695,
		1710,
		1995,
		2020,
		FDD,
		339000,
		342000,
		399000,
		404000,
		map[int][]int{
			15: {5, 10, 15, 20, 25},
			30: {10, 15, 20, 25},
			60: {10, 15, 20, 25},
		},
	},
	"n71": {
		"n71",
		663,
		698,
		617,
		652,
		FDD,
		132600,
		139600,
		123400,
		130400,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 35},
			30: {10, 15, 20, 25, 30, 35},
			60: {},
		},
	},
	"n72": {
		"n72",
		451,
		456,
		461,
		466,
		FDD,
		90200,
		91200,
		92200,
		93200,
		map[int][]int{
			15: {3, 5},
			30: {},
			60: {},
		},
	},
	"n74": {
		"n74",
		1427,
		1470,
		1475,
		1518,
		FDD,
		285400,
		294000,
		295000,
		303600,
		map[int][]int{
			15: {5, 10, 15, 20},
			30: {10, 15, 20},
			60: {10, 15, 20},
		},
	},
	"n75": {
		"n75",
		0,
		0,
		1432,
		1517,
		SDL,
		0,
		0,
		286400,
		303400,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 40, 50},
			30: {10, 15, 20, 25, 30, 40, 50},
			60: {10, 15, 20, 25, 30, 40, 50},
		},
	},
	"n76": {
		"n76",
		0,
		0,
		1427,
		1432,
		SDL,
		0,
		0,
		285400,
		286400,
		map[int][]int{
			15: {5},
			30: {},
			60: {},
		},
	},
	"n77": {
		"n77",
		3300,
		4200,
		3300,
		4200,
		TDD,
		620000,
		680000,
		620000,
		680000,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 40, 50},
			30: {10, 15, 20, 25, 30, 40, 50, 60, 70, 80, 90, 100},
			60: {10, 15, 20, 25, 30, 40, 50, 60, 70, 80, 90, 100},
		},
	},
	"n78": {
		"n78",
		3300,
		3800,
		3300,
		3800,
		TDD,
		620000,
		653333,
		620000,
		653333,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 40, 50},
			30: {10, 15, 20, 25, 30, 40, 50, 60, 70, 80, 90, 100},
			60: {10, 15, 20, 25, 30, 40, 50, 60, 70, 80, 90, 100},
		},
	},
	"n79": {
		"n79",
		4400,
		5000,
		4400,
		5000,
		TDD,
		693334,
		733333,
		693334,
		733333,
		map[int][]int{
			15: {10, 20, 30, 40, 50},
			30: {10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
			60: {10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
		},
	},
	"n80": {
		"n80",
		1710,
		1785,
		0,
		0,
		SUL,
		342000,
		357000,
		0,
		0,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 40},
			30: {10, 15, 20, 25, 30, 40},
			60: {10, 15, 20, 25, 30, 40},
		},
	},
	"n81": {
		"n81",
		880,
		915,
		0,
		0,
		SUL,
		176000,
		183000,
		0,
		0,
		map[int][]int{
			15: {5, 10, 15, 20},
			30: {10, 15, 20},
			60: {},
		},
	},
	"n82": {
		"n82",
		832,
		862,
		0,
		0,
		SUL,
		166400,
		172400,
		0,
		0,
		map[int][]int{
			15: {5, 10, 15, 20},
			30: {10, 15, 20},
			60: {},
		},
	},
	"n83": {
		"n83",
		703,
		748,
		0,
		0,
		SUL,
		140600,
		149600,
		0,
		0,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 40},
			30: {10, 15, 20, 25, 30, 40},
			60: {},
		},
	},
	"n84": {
		"n84",
		1920,
		1980,
		0,
		0,
		SUL,
		384000,
		396000,
		0,
		0,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 40, 50},
			30: {10, 15, 20, 25, 30, 40, 50},
			60: {10, 15, 20, 25, 30, 40, 50},
		},
	},
	"n85": {
		"n85",
		698,
		716,
		728,
		746,
		FDD,
		139600,
		143200,
		145600,
		149200,
		map[int][]int{
			15: {3, 5, 10, 15},
			30: {10, 15},
			60: {},
		},
	},
	"n86": {
		"n86",
		1710,
		1780,
		0,
		0,
		SUL,
		342000,
		356000,
		0,
		0,
		map[int][]int{
			15: {5, 10, 15, 20, 40},
			30: {10, 15, 20, 40},
			60: {10, 15, 20, 40},
		},
	},
	"n89": {
		"n89",
		824,
		849,
		0,
		0,
		SUL,
		164800,
		169800,
		0,
		0,
		map[int][]int{
			15: {5, 10, 15, 20},
			30: {10, 15, 20},
			60: {},
		},
	},
	"n90": {
		"n90",
		2496,
		2690,
		2496,
		2690,
		TDD,
		499200,
		537999,
		499200,
		537999,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 35, 40, 45, 50},
			30: {10, 15, 20, 25, 30, 35, 40, 45, 50, 60, 70, 80, 90, 100},
			60: {10, 15, 20, 25, 30, 35, 40, 45, 50, 60, 70, 80, 90, 100},
		},
	},
	"n91": {
		"n91",
		832,
		862,
		1427,
		1432,
		FDD,
		166400,
		172400,
		285400,
		286400,
		map[int][]int{
			15: {5, 10},
			30: {},
			60: {},
		},
	},
	"n92": {
		"n92",
		832,
		862,
		1432,
		1517,
		FDD,
		166400,
		172400,
		286400,
		303400,
		map[int][]int{
			15: {5, 10, 15, 20},
			30: {10, 15, 20},
			60: {},
		},
	},
	"n93": {
		"n93",
		880,
		915,
		1427,
		1432,
		FDD,
		176000,
		183000,
		285400,
		286400,
		map[int][]int{
			15: {5, 10},
			30: {},
			60: {},
		},
	},
	"n94": {
		"n94",
		880,
		915,
		1432,
		1517,
		FDD,
		176000,
		183000,
		286400,
		303400,
		map[int][]int{
			15: {5, 10, 15, 20},
			30: {10, 15, 20},
			60: {},
		},
	},
	"n95": {
		"n95",
		2010,
		2025,
		0,
		0,
		SUL,
		402000,
		405000,
		0,
		0,
		map[int][]int{
			15: {5, 10, 15},
			30: {10, 15},
			60: {10, 15},
		},
	},
	"n96": {
		"n96",
		5925,
		7125,
		5925,
		7125,
		TDD,
		795000,
		875000,
		795000,
		875000,
		map[int][]int{
			15: {20, 40},
			30: {20, 40, 60, 80, 100},
			60: {20, 40, 60, 80, 100},
		},
	},
	"n97": {
		"n97",
		2300,
		2400,
		0,
		0,
		SUL,
		460000,
		480000,
		0,
		0,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 40, 50},
			30: {10, 15, 20, 25, 30, 40, 50, 60, 70, 80, 90, 100},
			60: {10, 15, 20, 25, 30, 40, 50, 60, 70, 80, 90, 100},
		},
	},
	"n98": {
		"n98",
		1880,
		1920,
		0,
		0,
		SUL,
		376000,
		384000,
		0,
		0,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 35, 40},
			30: {10, 15, 20, 25, 30, 35, 40},
			60: {10, 15, 20, 25, 30, 35, 40},
		},
	},
	"n99": {
		"n99",
		1626.5,
		1660.5,
		0,
		0,
		SUL,
		325300,
		332100,
		0,
		0,
		map[int][]int{
			15: {5, 10},
			30: {10},
			60: {10},
		},
	},
	"n100": {
		"n100",
		874.4,
		880,
		919.4,
		925,
		FDD,
		174880,
		176000,
		183880,
		185000,
		map[int][]int{
			15: {3, 5},
			30: {},
			60: {},
		},
	},
	"n101": {
		"n101",
		1900,
		1910,
		1900,
		1910,
		TDD,
		380000,
		382000,
		380000,
		382000,
		map[int][]int{
			15: {5, 10},
			30: {10},
			60: {},
		},
	},
	"n102": {
		"n102",
		5925,
		6425,
		5925,
		6425,
		TDD,
		795000,
		828333,
		795000,
		828333,
		map[int][]int{
			15: {20, 40},
			30: {20, 40, 60, 80, 100},
			60: {20, 40, 60, 80, 100},
		},
	},
	"n104": {
		"n104",
		6425,
		7125,
		6425,
		7125,
		TDD,
		828334,
		875000,
		828334,
		875000,
		map[int][]int{
			15: {20, 30, 40, 50},
			30: {20, 30, 40, 50, 60, 70, 80, 90, 100},
			60: {20, 30, 40, 50, 60, 70, 80, 90, 100},
		},
	},
	"n105": {
		"n105",
		663,
		703,
		612,
		652,
		FDD,
		132600,
		140600,
		122400,
		130400,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 35},
			30: {10, 15, 20, 25, 30, 35},
			60: {},
		},
	},
	"n106": {
		"n106",
		896,
		901,
		935,
		940,
		FDD,
		179200,
		180200,
		187000,
		188000,
		map[int][]int{
			15: {3},
			30: {},
			60: {},
		},
	},
	"n109": {
		"n109",
		703,
		733,
		1432,
		1517,
		FDD,
		140600,
		146600,
		286400,
		303400,
		map[int][]int{
			15: {5, 10, 15, 20, 25, 30, 40, 50},
			30: {10, 15, 20, 25, 30, 40, 50},
			60: {},
		},
	},
	"n256": {
		"n256",
		1980,
		2010,
		2170,
		2200,
		FDD,
		396000,
		402000,
		434000,
		440000,
		map[int][]int{
			15: {5, 10, 15, 20},
			30: {10, 15, 20},
			60: {10, 15, 20},
		},
	},
	"n255": {
		"n255",
		1626.5,
		1660.5,
		1525,
		1559,
		FDD,
		325300,
		332100,
		305000,
		311800,
		map[int][]int{
			15: {5, 10, 15, 20},
			30: {10, 15, 20},
			60: {10, 15, 20},
		},
	},
	"n254": {
		"n254",
		1610,
		1626.5,
		2483.5,
		2500,
		FDD,
		322000,
		325300,
		496700,
		500000,
		map[int][]int{
			15: {5, 10, 15},
			30: {10, 15},
			60: {10, 15},
		},
	},
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

// GetBandNR takes a frequency and direction and returns the operating nr band name.
func GetBandNR(arfcn uint32, direction string) (nrBand BandNR, found bool) {
	arfcnFloat := float64(arfcn)
	for b := range BandsNR {
		band := BandsNR[b]

		found = direction == UL && band.ArfcnULlow <= arfcnFloat && arfcnFloat <= band.ArfcnULhigh
		if found {
			return band, found
		}

		found = direction == DL && band.ArfcnDLlow <= arfcnFloat && arfcnFloat <= band.ArfcnDLhigh
		if found {
			return band, found
		}
	}
	return
}

// GetBandEUTRA takes a frequency and direction and returns the operating EUTRA band name.
func GetBandEUTRA(earfcn uint32, direction string) (nrBand BandEutra, found bool) {
	arfcnFloat := float64(earfcn)
	for b := range BandsEutra {
		band := BandsEutra[b]

		found = direction == UL && band.EarfcnULlow <= arfcnFloat && arfcnFloat <= band.EarfcnULhigh
		if found {
			return band, found
		}

		found = direction == DL && band.EarfcnDLlow <= arfcnFloat && arfcnFloat <= band.EarfcnDLhigh
		if found {
			return band, found
		}
	}
	return
}

// GetFR takes a frequency and returns the frequency range designation.
func GetFR(arfcn float64) string {
	freq := CalculateFrequency(int(arfcn))
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

func CalculateFrequency(arfcn int) float64 {
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
type ChannelInfo struct {
	FR           string
	SCS          uint32
	ChannelBWMHz uint32 // Channel Bandwidth in MHz
}

var ChannelInfoByFR = map[ChannelInfo]uint32{
	{"FR1", 15, 5}:  25,
	{"FR1", 15, 10}: 52,
	{"FR1", 15, 15}: 79,
	{"FR1", 15, 20}: 106,
	{"FR1", 15, 25}: 106,
	{"FR1", 15, 30}: 160,
	{"FR1", 15, 35}: 188,
	{"FR1", 15, 40}: 216,
	{"FR1", 15, 45}: 242,
	{"FR1", 15, 50}: 270,

	{"FR1", 30, 5}:   11,
	{"FR1", 30, 10}:  24,
	{"FR1", 30, 15}:  38,
	{"FR1", 30, 20}:  51,
	{"FR1", 30, 25}:  51,
	{"FR1", 30, 30}:  78,
	{"FR1", 30, 35}:  92,
	{"FR1", 30, 40}:  106,
	{"FR1", 30, 45}:  119,
	{"FR1", 30, 50}:  133,
	{"FR1", 30, 60}:  162,
	{"FR1", 30, 70}:  189,
	{"FR1", 30, 80}:  217,
	{"FR1", 30, 90}:  245,
	{"FR1", 30, 100}: 273,

	{"FR1", 60, 10}:  11,
	{"FR1", 60, 15}:  18,
	{"FR1", 60, 20}:  24,
	{"FR1", 60, 25}:  24,
	{"FR1", 60, 30}:  38,
	{"FR1", 60, 35}:  44,
	{"FR1", 60, 40}:  51,
	{"FR1", 60, 45}:  58,
	{"FR1", 60, 50}:  65,
	{"FR1", 60, 60}:  79,
	{"FR1", 60, 70}:  93,
	{"FR1", 60, 80}:  107,
	{"FR1", 60, 90}:  121,
	{"FR1", 60, 100}: 136,

	{"FR2", 60, 50}:  66,
	{"FR2", 60, 100}: 132,
	{"FR2", 60, 200}: 264,

	{"FR2", 120, 50}:  32,
	{"FR2", 120, 100}: 66,
	{"FR2", 120, 200}: 132,
	{"FR2", 120, 400}: 264,

	{"FR2", 480, 400}:  66,
	{"FR2", 480, 800}:  124,
	{"FR2", 480, 1600}: 248,

	{"FR2", 960, 400}:  33,
	{"FR2", 960, 800}:  61,
	{"FR2", 960, 1600}: 124,
	{"FR2", 960, 2000}: 148,
}

// GetPRBs returns the PRBs for the available Channel Bandwidth and SCS.
func GetPRBs(channelBW uint32, scs int, fr string) int {
	if (fr == FR1 && channelBW < 5) || (fr == FR2 && channelBW < 50) {
		return 0
	}

	ci := ChannelInfo{
		FR:           fr,
		SCS:          uint32(scs),
		ChannelBWMHz: uint32(roundToNearestChannelBW(fr, int(channelBW))),
	}
	return int(ChannelInfoByFR[ci])
}

func roundToNearestChannelBW(fr string, chBw int) int {
	roundFactor := 0.0
	switch fr {
	case FR1:
		{
			switch {
			case chBw >= 50:
				roundFactor = 10.0
			default:
				roundFactor = 5.0
			}
		}
	case FR2:
		{
			switch {
			case chBw < 100:
				roundFactor = 50.0
			case chBw < 200:
				roundFactor = 100.0
			case chBw < 400:
				roundFactor = 200.0
			default:
				roundFactor = 400.0
			}
		}
	}
	return int(math.Round(float64(chBw)/roundFactor) * roundFactor)
}

type OHKey struct {
	FR        string
	Direction string
}

// 3GPP TS 38.306
// 4.1.2 Supported max data rate for DL/UL
var OH = map[OHKey]float64{
	{FR1, DL}: 0.14,
	{FR2, DL}: 0.18,
	{FR1, UL}: 0.08,
	{FR2, UL}: 0.10,
}
