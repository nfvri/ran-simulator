package bandwidth

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"

	"github.com/nfvri/onos-api/go/onos/ransim/types"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/utils"
	art "github.com/plar/go-adaptive-radix-tree/v2"
	"github.com/sirupsen/logrus"
)

type CAConfigNR struct {
	Name           string
	BWComboSet     int
	Bands          []BandNR
	AggregateBWMHz int
}

var caBandCombinationsNR = []string{
	"n1_n3",
	"n1_n5",
	"n1_n7",
	"n1_n8",
	"n1_n18",
	"n1_n20",
	"n1_n26",
	"n1_n28",
	"n1_n38",
	"n1_n40",
	"n1_n41",
	"n1_n46",
	"n1_n67",
	"n1_n74",
	"n1_n75",
	"n1_n77",
	"n1_n78",
	"n1_n79",
	"n1_n102",
	"n2_n5",
	"n2_n7",
	"n2_n12",
	"n2_n14",
	"n2_n29",
	"n2_n30",
	"n2_n38",
	"n2_n41",
	"n2_n48",
	"n2_n66",
	"n2_n71",
	"n2_n77",
	"n2_n78",
	"n3_n5",
	"n3_n7",
	"n3_n8",
	"n3_n18",
	"n3_n20",
	"n3_n26",
	"n3_n28",
	"n3_n34",
	"n3_n38",
	"n3_n40",
	"n3_n41",
	"n3_n67",
	"n3_n74",
	"n3_n75",
	"n3_n77",
	"n3_n78",
	"n3_n79",
	"n5_n7",
	"n5_n12",
	"n5_n14",
	"n5_n25",
	"n5_n28",
	"n5_n29",
	"n5_n30",
	"n5_n40",
	"n5_n48",
	"n5_n66",
	"n5_n77",
	"n5_n78",
	"n5_n79",
	"n7_n8",
	"n7_n25",
	"n7_n26",
	"n7_n28",
	"n7_n40",
	"n7_n46",
	"n7_n66",
	"n7_n67",
	"n7_n75",
	"n7_n77",
	"n7_n78",
	"n7_n79",
	"n7_n102",
	"n8_n20",
	"n8_n28",
	"n8_n34",
	"n8_n38",
	"n8_n39",
	"n8_n40",
	"n8_n41",
	"n8_n75",
	"n8_n77",
	"n8_n78",
	"n8_n79",
	"n12_n25",
	"n12_n30",
	"n12_n48",
	"n12_n66",
	"n12_n71",
	"n12_n77",
	"n13_n25",
	"n13_n66",
	"n13_n77",
	"n14_n30",
	"n14_n66",
	"n14_n77",
	"n18_n28",
	"n18_n40",
	"n18_n41",
	"n18_n74",
	"n18_n77",
	"n18_n78",
	"n20_n28",
	"n20_n40",
	"n20_n67",
	"n20_n75",
	"n20_n78",
	"n24_n41",
	"n24_n48",
	"n24_n77",
	"n25_n29",
	"n25_n38",
	"n25_n41",
	"n25_n46",
	"n25_n48",
	"n25_n66",
	"n25_n71",
	"n25_n77",
	"n25_n78",
	"n25_n85",
	"n26_n28",
	"n26_n66",
	"n26_n70",
	"n26_n77",
	"n26_n78",
	"n28_n34",
	"n28_n38",
	"n28_n39",
	"n28_n40",
	"n28_n41",
	"n28_n46",
	"n28_n50",
	"n28_n71",
	"n28_n74",
	"n28_n75",
	"n28_n77",
	"n28_n78",
	"n28_n79",
	"n28_n94",
	"n28_n102",
	"n29_n30",
	"n29_n66",
	"n29_n70",
	"n29_n71",
	"n29_n77",
	"n30_n66",
	"n30_n77",
	"n34_n40",
	"n34_n41",
	"n34_n79",
	"n38_n40",
	"n38_n66",
	"n38_n71",
	"n38_n78",
	"n38_n79",
	"n39_n40",
	"n39_n41",
	"n39_n79",
	"n40_n41",
	"n40_n77",
	"n40_n78",
	"n40_n79",
	"n41_n48",
	"n41_n50",
	"n41_n66",
	"n41_n70",
	"n41_n71",
	"n41_n74",
	"n41_n77",
	"n41_n78",
	"n41_n79",
	"n41_n85",
	"n46_n48",
	"n46_n66",
	"n46_n78",
	"n46_n96",
	"n48_n53",
	"n48_n66",
	"n48_n70",
	"n48_n71",
	"n48_n77",
	"n48_n96",
	"n50_n78",
	"n66_n70",
	"n66_n71",
	"n66_n77",
	"n66_n78",
	"n66_n85",
	"n67_n78",
	"n70_n71",
	"n70_n77",
	"n70_n78",
	"n71_n77",
	"n71_n78",
	"n74_n77",
	"n74_n78",
	"n75_n78",
	"n76_n78",
	"n77_n78",
	"n77_n79",
	"n77_n85",
	"n78_n79",
	"n78_n92",
	"n78_n94",
	"n78_n102",
	"n1_n3_n5",
	"n1_n3_n7",
	"n1_n3_n8",
	"n1_n3_n18",
	"n1_n3_n20",
	"n1_n3_n26",
	"n1_n3_n28",
	"n1_n3_n38",
	"n1_n3_n40",
	"n1_n3_n41",
	"n1_n3_n67",
	"n1_n3_n77",
	"n1_n3_n78",
	"n1_n3_n79",
	"n1_n5_n7",
	"n1_n5_n28",
	"n1_n5_n78",
	"n1_n7_n8",
	"n1_n7_n26",
	"n1_n7_n28",
	"n1_n7_n38",
	"n1_n7_n40",
	"n1_n7_n78",
	"n1_n7_n79",
	"n1_n8_n28",
	"n1_n8_n40",
	"n1_n8_n77",
	"n1_n8_n78",
	"n1_n8_n79",
	"n1_n18_n28",
	"n1_n18_n41",
	"n1_n18_n77",
	"n1_n20_n67",
	"n1_n20_n78",
	"n1_n26_n78",
	"n1_n28_n38",
	"n1_n28_n40",
	"n1_n28_n41",
	"n1_n28_n77",
	"n1_n28_n78",
	"n1_n28_n79",
	"n1_n38_n78",
	"n1_n40_n77",
	"n1_n40_n78",
	"n1_n41_n77",
	"n1_n41_n79",
	"n1_n77_n79",
	"n1_n78_n79",
	"n2_n5_n30",
	"n2_n5_n48",
	"n2_n5_n66",
	"n2_n5_n77",
	"n2_n12_n30",
	"n2_n12_n66",
	"n2_n12_n77",
	"n2_n14_n30",
	"n2_n14_n66",
	"n2_n14_n77",
	"n2_n29_n30",
	"n2_n29_n66",
	"n2_n29_n77",
	"n2_n30_n66",
	"n2_n30_n77",
	"n2_n48_n66",
	"n2_n48_n77",
	"n2_n66_n71",
	"n2_n66_n77",
	"n2_n66_n78",
	"n2_n71_n78",
	"n3_n5_n7",
	"n3_n5_n28",
	"n3_n5_n78",
	"n3_n7_n8",
	"n3_n7_n26",
	"n3_n7_n28",
	"n3_n7_n38",
	"n3_n7_n67",
	"n3_n7_n78",
	"n3_n7_n79",
	"n3_n8_n28",
	"n3_n8_n41",
	"n3_n8_n77",
	"n3_n8_n78",
	"n3_n18_n28",
	"n3_n18_n41",
	"n3_n18_n77",
	"n3_n20_n28",
	"n3_n20_n67",
	"n3_n20_n78",
	"n3_n26_n78",
	"n3_n28_n38",
	"n3_n28_n40",
	"n3_n28_n41",
	"n3_n28_n77",
	"n3_n28_n78",
	"n3_n28_n79",
	"n3_n38_n40",
	"n3_n40_n41",
	"n3_n40_n77",
	"n3_n41_n77",
	"n3_n41_n78",
	"n3_n41_n79",
	"n3_n67_n78",
	"n3_n77_n79",
	"n3_n78_n79",
	"n5_n7_n28",
	"n5_n7_n77",
	"n5_n7_n78",
	"n5_n12_n77",
	"n5_n14_n77",
	"n5_n25_n66",
	"n5_n25_n77",
	"n5_n25_n78",
	"n5_n29_n77",
	"n5_n30_n66",
	"n5_n30_n77",
	"n5_n40_n78",
	"n5_n48_n66",
	"n5_n48_n77",
	"n5_n66_n77",
	"n5_n66_n78",
	"n7_n8_n28",
	"n7_n8_n40",
	"n7_n8_n78",
	"n7_n25_n66",
	"n7_n25_n77",
	"n7_n25_n78",
	"n7_n26_n78",
	"n7_n28_n38",
	"n7_n28_n78",
	"n7_n40_n78",
	"n7_n46_n78",
	"n7_n66_n77",
	"n7_n66_n78",
	"n7_n71_n77",
	"n8_n28_n78",
	"n8_n38_n40",
	"n8_n39_n41",
	"n8_n39_n79",
	"n8_n40_n41",
	"n8_n40_n78",
	"n8_n41_n79",
	"n8_n78_n79",
	"n12_n30_n66",
	"n12_n30_n77",
	"n12_n66_n77",
	"n13_n25_n66",
	"n13_n25_n77",
	"n13_n66_n77",
	"n14_n30_n66",
	"n14_n30_n77",
	"n14_n66_n77",
	"n18_n28_n41",
	"n18_n28_n77",
	"n18_n41_n77",
	"n20_n28_n78",
	"n24_n41_n48",
	"n24_n41_n77",
	"n24_n48_n77",
	"n25_n29_n66",
	"n25_n38_n66",
	"n25_n38_n78",
	"n25_n41_n66",
	"n25_n41_n71",
	"n25_n41_n77",
	"n25_n41_n78",
	"n25_n48_n66",
	"n25_n66_n71",
	"n25_n66_n77",
	"n25_n66_n78",
	"n25_n71_n77",
	"n25_n71_n78",
	"n26_n66_n70",
	"n28_n38_n78",
	"n28_n39_n40",
	"n28_n39_n41",
	"n28_n39_n79",
	"n28_n40_n41",
	"n28_n40_n77",
	"n28_n40_n78",
	"n28_n40_n79",
	"n28_n41_n77",
	"n28_n41_n78",
	"n28_n41_n79",
	"n28_n46_n78",
	"n28_n77_n79",
	"n28_n78_n79",
	"n29_n30_n66",
	"n29_n30_n77",
	"n29_n66_n70",
	"n29_n66_n77",
	"n29_n70_n71",
	"n30_n66_n77",
	"n38_n66_n78",
	"n39_n40_n41",
	"n39_n40_n79",
	"n39_n41_n79",
	"n40_n41_n79",
	"n41_n66_n70",
	"n41_n66_n71",
	"n41_n66_n77",
	"n41_n66_n78",
	"n41_n70_n78",
	"n41_n71_n77",
	"n41_n71_n78",
	"n41_n77_n79",
	"n46_n48_n96",
	"n48_n66_n70",
	"n48_n66_n71",
	"n48_n66_n77",
	"n48_n70_n71",
	"n48_n70_n77",
	"n48_n71_n77",
	"n66_n70_n71",
	"n66_n70_n77",
	"n66_n70_n78",
	"n66_n71_n77",
	"n66_n71_n78",
	"n70_n71_n77",
	"n1_n3_n5_n7",
	"n1_n3_n5_n78",
	"n1_n3_n7_n8",
	"n1_n3_n7_n26",
	"n1_n3_n7_n28",
	"n1_n3_n7_n38",
	"n1_n3_n7_n78",
	"n1_n3_n8_n77",
	"n1_n3_n8_n78",
	"n1_n3_n18_n28",
	"n1_n3_n18_n41",
	"n1_n3_n18_n77",
	"n1_n3_n26_n78",
	"n1_n3_n28_n38",
	"n1_n3_n28_n41",
	"n1_n3_n28_n77",
	"n1_n3_n28_n78",
	"n1_n3_n28_n79",
	"n1_n3_n38_n78",
	"n1_n3_n40_n77",
	"n1_n3_n41_n77",
	"n1_n3_n41_n79",
	"n1_n3_n77_n79",
	"n1_n5_n7_n78",
	"n1_n7_n8_n40",
	"n1_n7_n8_n78",
	"n1_n7_n26_n78",
	"n1_n7_n28_n38",
	"n1_n7_n28_n78",
	"n1_n7_n38_n78",
	"n1_n7_n40_n78",
	"n1_n8_n40_n78",
	"n1_n8_n78_n79",
	"n1_n18_n28_n41",
	"n1_n18_n28_n77",
	"n1_n18_n41_n77",
	"n1_n28_n38_n78",
	"n1_n28_n40_n77",
	"n1_n28_n40_n78",
	"n1_n28_n41_n77",
	"n1_n28_n41_n79",
	"n1_n28_n77_n79",
	"n1_n41_n77_n79",
	"n2_n5_n30_n66",
	"n2_n5_n30_n77",
	"n2_n5_n48_n66",
	"n2_n5_n48_n77",
	"n2_n5_n66_n77",
	"n2_n12_n30_n66",
	"n2_n12_n30_n77",
	"n2_n12_n66_n77",
	"n2_n14_n30_n66",
	"n2_n14_n30_n77",
	"n2_n14_n66_n77",
	"n2_n29_n30_n66",
	"n2_n29_n30_n77",
	"n2_n29_n66_n77",
	"n2_n30_n66_n77",
	"n2_n48_n66_n77",
	"n2_n66_n71_n78",
	"n3_n5_n7_n78",
	"n3_n7_n8_n78",
	"n3_n7_n26_n78",
	"n3_n7_n28_n38",
	"n3_n7_n28_n78",
	"n3_n7_n38_n78",
	"n3_n18_n28_n41",
	"n3_n18_n28_n77",
	"n3_n18_n41_n77",
	"n3_n28_n38_n78",
	"n3_n28_n40_n77",
	"n3_n28_n41_n77",
	"n3_n28_n41_n78",
	"n3_n28_n41_n79",
	"n3_n28_n77_n79",
	"n3_n41_n77_n79",
	"n5_n25_n66_n77",
	"n5_n25_n66_n78",
	"n5_n30_n66_n77",
	"n5_n48_n66_n77",
	"n7_n8_n40_n78",
	"n7_n25_n66_n77",
	"n7_n25_n66_n78",
	"n7_n28_n38_n78",
	"n12_n30_n66_n77",
	"n13_n25_n66_n77",
	"n14_n30_n66_n77",
	"n18_n28_n41_n77",
	"n25_n38_n66_n78",
	"n25_n41_n66_n71",
	"n25_n41_n66_n77",
	"n25_n41_n66_n78",
	"n25_n41_n71_n77",
	"n25_n41_n71_n78",
	"n25_n66_n71_n77",
	"n25_n66_n71_n78",
	"n28_n41_n77_n79",
	"n29_n30_n66_n77",
	"n41_n66_n70_n78",
	"n41_n66_n71_n77",
	"n41_n66_n71_n78",
	"n48_n66_n70_n71",
	"n48_n66_n70_n77",
	"n48_n66_n71_n77",
	"n48_n70_n71_n77",
	"n66_n70_n71_n77",
	"n1_n3_n5_n7_n78",
	"n1_n3_n7_n8_n78",
	"n1_n3_n7_n26_n78",
	"n1_n3_n7_n28_n38",
	"n1_n3_n7_n28_n78",
	"n1_n3_n7_n38_n78",
	"n1_n3_n28_n38_n78",
	"n1_n3_n28_n41_n77",
	"n1_n3_n28_n41_n79",
	"n1_n3_n28_n77_n79",
	"n1_n3_n41_n77_n79",
	"n1_n7_n28_n38_n78",
	"n1_n28_n41_n77_n79",
	"n2_n5_n30_n66_n77",
	"n2_n5_n48_n66_n77",
	"n2_n12_n30_n66_n77",
	"n2_n14_n30_n66_n77",
	"n2_n29_n30_n66_n77",
	"n3_n7_n28_n38_n78",
	"n3_n28_n41_n77_n79",
	"n25_n41_n66_n71_n77",
	"n1_n3_n7_n28_n38_n78",
}

type CarrierAggregatorNR struct {
	bandCombinationsNRTree art.Tree
}

func NewCarrierAggregatorNR() (ca *CarrierAggregatorNR) {
	caNR := &CarrierAggregatorNR{
		bandCombinationsNRTree: art.New(),
	}
	for _, combination := range caBandCombinationsNR {
		caNR.bandCombinationsNRTree.Insert(art.Key(combination), true)
	}
	return caNR
}

func (ca *CarrierAggregatorNR) IsValidBandCombination(bandCombo string) bool {
	_, found := ca.bandCombinationsNRTree.Search(art.Key(bandCombo))
	return found
}

func (ca *CarrierAggregatorNR) GetValidCACombinations(targetCells []*model.Cell) (validCABandCombos [][]string, cellsByBand map[string][]*model.Cell) {

	validCABandCombos = make([][]string, 0)
	cellsByBand = make(map[string][]*model.Cell)
	for cellIndex := range targetCells {
		cell := targetCells[cellIndex]
		arfcn := utils.If(cell.ArfcnDL > 0, cell.ArfcnDL, cell.ArfcnUL)
		direction := utils.If(cell.ArfcnDL > 0, DL, UL)
		cellBand, found := GetBand(arfcn, direction)
		if !found {
			continue
		}
		cellsByBand[cellBand.Name] = append(cellsByBand[cellBand.Name], cell)
	}

	targetCellBands := []string{}
	for b := range cellsByBand {
		targetCellBands = append(targetCellBands, b)
	}

	// TODO: also add intra-band CA combinations
	// FIXME: For cells on different nodes
	// we should consult DUAL CONNECTIVITY combinations in:
	// https://www.etsi.org/deliver/etsi_ts/138100_138199/13810103/15.02.00_60/ts_13810103v150200p.pdf
	// https://www.sqimway.com/nr_nrdc.php
	// https://www.sqimway.com/nr_endc.php
	// https://www.sqimway.com/nr_nedc.php
	caBandCombos := GetCABandCombinations(SortNRBands(targetCellBands))

	for _, caBandCombo := range caBandCombos {
		if ca.IsValidBandCombination(caBandCombo) {
			validCABandCombos = append(validCABandCombos, strings.Split(caBandCombo, "_"))
		}
	}

	return

}

// Function to sort NR bands based on their names
func SortNRBands(bands []string) []string {
	bandNumber := func(s string) int {
		num, _ := strconv.Atoi(s[1:])
		return num
	}
	sort.Slice(bands, func(i, j int) bool {
		return bandNumber(bands[i]) < bandNumber(bands[j])
	})
	return bands
}

func GetCABandCombinations(sortedBandsNR []string) []string {
	logrus.Info("[GetCABandCombinations]...")
	var combinations []string
	queue := []string{}

	for i := 0; i < len(sortedBandsNR); i++ {
		for j := i + 1; j < len(sortedBandsNR); j++ {
			initialCombo := sortedBandsNR[i] + "_" + sortedBandsNR[j]
			queue = append(queue, initialCombo)
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		combinations = append(combinations, current)

		for _, b := range sortedBandsNR {
			if !strings.Contains(current, b) {
				newCombination := current + "_" + b
				if !slices.Contains(combinations, newCombination) {
					queue = append(queue, newCombination)
				}
			}
		}
	}

	return combinations
}

type prbInfo struct {
	prbsUL int
	prbsDL int
}
type CAScheme struct {
	Bands            []string
	AvailPRBsPerCell map[types.NCGI]prbInfo
	FixedAllocCells  []*model.Cell
	ReallocCells     []*model.Cell
	CanBeImplemented bool
}

// GetFeasibleCASchemes searches for cells operating at
// valid band combinations for CA and provide sufficient bandwidth
// for the UE.
func GetFeasibleCASchemes(
	validCABandCombos [][]string,
	cellsByBand map[string][]*model.Cell,
	ranModel *model.Model,
	ue *model.UE) []CAScheme {

	logrus.Info("[GetFeasibleCASchemes]...")
	ueRequiredPRBsDL, ueRequiredPRBsUL := CurrPRBsUsed(ue)

	maxPRBsDL := 0
	var maxBWCAScheme CAScheme
	feasibleCASchemes := []CAScheme{}

	for _, bandCombo := range validCABandCombos {
		var bandComboCells []*model.Cell
		for _, band := range bandCombo {
			if cells, exists := cellsByBand[band]; exists {
				bandComboCells = append(bandComboCells, cells...)
			} else {
				continue
			}
		}

		comboAvailPRBsDL, comboAvailPRBsUL, availPRBsPerCell := GetCAComboAvailPRBs(bandComboCells, ranModel, ue)

		if comboAvailPRBsDL > ueRequiredPRBsUL && comboAvailPRBsUL > ueRequiredPRBsDL {
			feasibleCASchemes = append(feasibleCASchemes, CAScheme{
				Bands:            bandCombo,
				FixedAllocCells:  bandComboCells,
				AvailPRBsPerCell: availPRBsPerCell,
				CanBeImplemented: true,
			})
		}

		if comboAvailPRBsDL > maxPRBsDL {
			maxPRBsDL = comboAvailPRBsDL
			maxBWCAScheme = CAScheme{
				Bands:            bandCombo,
				FixedAllocCells:  bandComboCells[:3],
				ReallocCells:     bandComboCells[3:],
				AvailPRBsPerCell: availPRBsPerCell,
				CanBeImplemented: true,
			}
		}
	}

	anyFeasibleCAScheme := len(feasibleCASchemes) > 0
	if !anyFeasibleCAScheme && len(maxBWCAScheme.Bands) > 0 {
		feasibleCASchemes = append(feasibleCASchemes, maxBWCAScheme)
	}

	return feasibleCASchemes
}

func calculateMeanPRBs(prbMap map[types.NCGI]prbInfo) float64 {
	totalPRBs := 0
	numCells := len(prbMap)
	for _, prb := range prbMap {
		totalPRBs += (prb.prbsUL + prb.prbsDL)
	}
	return float64(totalPRBs) / float64(numCells)
}

func calculatePRBVariance(prbMap map[types.NCGI]prbInfo, meanPRBs float64) map[types.NCGI]float64 {
	varianceMap := make(map[types.NCGI]float64)
	for ncgi, prb := range prbMap {
		totalPRBs := float64(prb.prbsUL + prb.prbsDL)
		varianceMap[ncgi] = math.Abs(totalPRBs - meanPRBs)
	}
	return varianceMap
}

func ChooseReallocCells(bandComboCells []*model.Cell, availPRBsPerCell map[types.NCGI]prbInfo) (fixedAllocationSet, dynamicReallocationSet []*model.Cell) {

	// allocate available in descending order of avail PRBS
	// and for the rest reallocate
	// use threshold %
	// and maybe TDD/FDD mode of operating band
	// e.g. reallocate for TDD to be in sync or time share

	// Calculate Mean PRB Availability
	meanPRBs := calculateMeanPRBs(availPRBsPerCell)
	prbVariance := calculatePRBVariance(availPRBsPerCell, meanPRBs)

	fixedAllocationSet = make([]*model.Cell, 0)
	dynamicReallocationSet = make([]*model.Cell, 0)

	for _, cell := range bandComboCells {
		arfcn := utils.If(cell.Channel.ArfcnDL > 0, cell.Channel.ArfcnDL, cell.Channel.ArfcnUL)
		direction := utils.If(cell.Channel.ArfcnDL > 0, DL, UL)
		band, found := GetBand(arfcn, direction)
		if !found {
			continue // Skip if band info is not found
		}

		prb := availPRBsPerCell[cell.NCGI]
		totalPRBs := prb.prbsUL + prb.prbsDL
		variance := prbVariance[cell.NCGI]

		if band.DuplexingMode == "FDD" && totalPRBs >= int(meanPRBs) {
			fixedAllocationSet = append(fixedAllocationSet, cell)
		} else if band.DuplexingMode == "TDD" && variance > meanPRBs*0.2 {
			// If variance is high, prioritize dynamic allocation for TDD
			dynamicReallocationSet = append(dynamicReallocationSet, cell)
		} else {
			fixedAllocationSet = append(fixedAllocationSet, cell)
		}
	}

	return
}

// GetCAComboAvailPRBs returns the total number of available PRBs in DL and UL
// for the cell CA combination and also the per cell available PRBs.
func GetCAComboAvailPRBs(bandComboCells []*model.Cell, ranModel *model.Model, ue *model.UE) (int, int, map[types.NCGI]prbInfo) {
	comboAvailPRBsDL := 0
	comboAvailPRBsUL := 0
	availPRBsPerCell := map[types.NCGI]prbInfo{}

	for _, cell := range bandComboCells {

		servedUEs := ranModel.GetServedUEs(cell.NCGI)
		cellAvailPrbsUL, cellAvailPrbsDL, err := GetCellAvailPRBs(cell, servedUEs, ue)
		if err != nil {
			continue
		}

		availPRBsPerCell[cell.NCGI] = prbInfo{
			prbsUL: cellAvailPrbsUL,
			prbsDL: cellAvailPrbsDL,
		}
		comboAvailPRBsUL += cellAvailPrbsUL
		comboAvailPRBsDL += cellAvailPrbsDL

	}
	return comboAvailPRBsDL, comboAvailPRBsUL, availPRBsPerCell
}

// GetCellAvailPRBs returns the available PRBs for the provided cell
// given its served UEs.
func GetCellAvailPRBs(cell *model.Cell, servedUEs []*model.UE, ue *model.UE) (int, int, error) {

	usedBWDL := 0.0
	usedBWUL := 0.0

	for _, servedUE := range servedUEs {
		// dont count already reserved bwps by ue
		if ue.IMSI == servedUE.IMSI {
			continue
		}
		ueServCell, _ := servedUE.GetServingCell(cell.NCGI)
		for _, bwp := range ueServCell.BwpRefs {
			if bwp.Downlink {
				usedBWDL += float64(bwp.NumberOfRBs) * float64(bwp.Scs) * 12
			} else {
				usedBWUL += float64(bwp.NumberOfRBs) * float64(bwp.Scs) * 12
			}
		}
	}

	arfcn := utils.If(cell.Channel.ArfcnDL > 0, float64(cell.Channel.ArfcnDL), float64(cell.Channel.ArfcnUL))
	fr := GetFR(arfcn)
	scs := NrSCSByCQIPerFR[fr][ue.FiveQi]
	cellAvailBwDL := float64(cell.Channel.BsChannelBwDL) - usedBWDL
	cellAvailBwUL := float64(cell.Channel.BsChannelBwUL) - usedBWUL

	cellAvailPrbsUL, err := GetPRBs(cellAvailBwUL, scs, fr)
	if err != nil {
		return -1, -1, err
	}
	cellAvailPrbsDL, err := GetPRBs(cellAvailBwDL, scs, fr)
	if err != nil {
		return -1, -1, err
	}
	return cellAvailPrbsUL, cellAvailPrbsDL, nil
}
