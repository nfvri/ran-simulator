package ues

import (
	"context"
	"strings"

	"math"
	"math/rand"
	"strconv"
	"sync"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/nfvri/onos-api/go/onos/ransim/metrics"
	"github.com/nfvri/onos-api/go/onos/ransim/types"
	bw "github.com/nfvri/ran-simulator/pkg/bandwidth"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/signal"
	redisLib "github.com/nfvri/ran-simulator/pkg/store/redis"
	"github.com/nfvri/ran-simulator/pkg/utils"
	mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-mho-go"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
)

// TODO: Decouple metrics, ueCreation and bwp Allocation. Use metrics in both initUeInStoreandServingMapings and initBWPs
// InitUEs  -> initMetrics
//
//	-> initUeInStoreandServingMapings
//	-> initBWPs
func InitUEs(cellMeasurements []*metrics.Metric, cells map[string]*model.Cell, cacheStore redisLib.Store, snapshotId string, dc, ueHeight float64) (map[string]*model.UE, bool) {

	numUEsByCell, prbMeasPerCell := bw.UtilizationInfoByCell(cellMeasurements)
	numUEsPerCQIByCell := bw.GetNumUEsPerCQIByCell(numUEsByCell)
	usedPRBsDLPerCQIByCell, usedPRBsULPerCQIByCell := bw.GetUsedPRBsPerCQIByCell(prbMeasPerCell, numUEsPerCQIByCell)

	for _, usedPRBsDL := range usedPRBsDLPerCQIByCell {
		sum := 0
		for _, numPRBs := range usedPRBsDL {
			sum += numPRBs
		}
	}
	for _, usedPRBsUL := range usedPRBsULPerCQIByCell {
		sum := 0
		for _, numPRBs := range usedPRBsUL {
			sum += numPRBs
		}
	}
	ues := map[string]*model.UE{}

	ctx := context.Background()
	ueGroup, err := cacheStore.GetUEGroup(ctx, snapshotId)
	storeInCache := snapshotId != "" && err != nil

	if err == nil {
		for imsi := range ueGroup {
			ue := ueGroup[imsi]
			ues[imsi] = &ue
		}
		return ues, storeInCache
	}

	for sCellNCGI, numUEsPerCQI := range numUEsPerCQIByCell {

		sCell, ok := cells[strconv.FormatUint(sCellNCGI, 10)]
		if !ok {
			continue
		}

		numUEsPerBeamQS := bw.GetNumUEsPerBeamQS(sCell, numUEsPerCQI)

		totalUEs := 0
		for _, numUEs := range numUEsPerBeamQS {
			totalUEs += numUEs
		}
		if totalUEs == 0 {
			log.Warnf("number of generated ues for cell %v is 0", sCellNCGI)
			continue
		}

		cellServedUEs := []*model.UE{}
		ues, cellServedUEs = GenerateUEsBasedOnBeamQS(sCell, cells, numUEsPerBeamQS, ueHeight, dc, prbMeasPerCell, ues)

		cellUsedPRBsDlPerCQI := usedPRBsDLPerCQIByCell[sCellNCGI]
		cellUsedPRBsUlPerCQI := usedPRBsULPerCQIByCell[sCellNCGI]
		availPRBsDL := prbMeasPerCell[sCellNCGI][bw.AVAIL_PRBS_DL_METRIC]
		availPRBsUL := prbMeasPerCell[sCellNCGI][bw.AVAIL_PRBS_UL_METRIC]
		log.Infof("cell:%v , cellServedUEs: %+v", sCell.NCGI, cellServedUEs)

		// FIXME: decide how to allocate allocate for all ue.ServingCells
		bw.InitBWPs(sCell, numUEsPerCQI, cellUsedPRBsDlPerCQI, cellUsedPRBsUlPerCQI, availPRBsDL, availPRBsUL, cellServedUEs)
	}

	log.Infof("------------- len(ues): %d --------------", len(ues))
	log.Infof("---------------- Updated UEs -----------------")
	return ues, storeInCache
}

func GenerateUEsBasedOnBeamQS(sCell *model.Cell, cells map[string]*model.Cell, numUEsPerBeamQS map[model.BeamQS]int, ueHeight float64, dc float64, prbMeasPerCell map[uint64]map[string]int, ues map[string]*model.UE) (map[string]*model.UE, []*model.UE) {
	nCells := utils.GetNeighborCells(sCell, cells, utils.By.FreqOrLocation)
	nBeamIDs := signal.GetNeighborBeamIDs(nCells)

	cellServedUEs := []*model.UE{}
	mtx := sync.RWMutex{}
	var wg sync.WaitGroup

	for beamQS, numUEs := range numUEsPerBeamQS {
		if numUEs <= 0 {
			continue
		}
		wg.Add(1)
		go func(servCell *model.Cell, beamQs model.BeamQS, numUes int) {
			defer wg.Done()
			ueSINR := signal.GetSINR(beamQs.CQI)
			ueLocations := signal.GetSinrPoints(servCell, beamQs.BeamID, nCells, nBeamIDs, ueHeight, ueSINR, dc, numUes, beamQs.CQI)
			ueRSRPs := GetUERsrpsBasedOnLocation(servCell, beamQs.BeamID, ueLocations, cells, ueHeight)

			for i := 0; i < numUes; i++ {
				if len(ueLocations) <= i {
					log.Error("number of ue locations generated is smaller than the required")
					break
				}

				ueRSRP := ueRSRPs[i]
				ueLocation := ueLocations[i]

				ueInterferingBeams := findInterferingBeams(ueLocation, servCell, beamQs.BeamID, cells, ueHeight, prbMeasPerCell)
				neighborCells := map[types.NCGI]*model.UECell{}
				for b := range ueInterferingBeams {
					beam := ueInterferingBeams[b]
					_, neighborExists := neighborCells[beam.NCGI]
					if !neighborExists || beam.Rsrp > neighborCells[beam.NCGI].Rsrp {
						neighborCells[beam.NCGI] = beam
					}
				}
				totalPrbsDl := prbMeasPerCell[uint64(servCell.NCGI)][bw.AVAIL_PRBS_DL_METRIC]
				ueRSRQ := math.Round(signal.RSRQ(ueSINR, totalPrbsDl)*100) / 100

				mtx.Lock()
				counter := len(ues) + 1
				mtx.Unlock()

				simUE, ueIMSI := CreateSimulationUE(
					uint64(servCell.NCGI),
					nCells,
					beamQs,
					counter,
					totalPrbsDl,
					ueHeight,
					ueSINR,
					ueRSRP,
					ueRSRQ,
					ueLocation,
					maps.Values(neighborCells),
					ueInterferingBeams,
				)

				mtx.Lock()
				ues[ueIMSI] = simUE
				cellServedUEs = append(cellServedUEs, simUE)
				mtx.Unlock()
			}
		}(sCell, beamQS, numUEs)
	}
	wg.Wait()
	return ues, cellServedUEs
}

func GetUERsrpsBasedOnLocation(sCell *model.Cell, beamID model.BeamID, uesLocations []model.Coordinate, cells map[string]*model.Cell, ueHeight float64) (ueRSRPs []float64) {

	ueRSRPs = []float64{}
	mpf := signal.RiceanFading(signal.GetRiceanK(sCell.GetCarrier(beamID)))

	for _, ueCoord := range uesLocations {
		rsrp := signal.Strength(ueCoord, ueHeight, mpf, sCell, beamID)
		ueRSRPs = append(ueRSRPs, math.Round(rsrp*100)/100)
	}

	return
}

func CreateSimulationUE(
	ncgi uint64,
	nCells map[types.NCGI]*model.Cell,
	beamQS model.BeamQS,
	counter, totalPrbsDl int,
	ueHeight, sinr, rsrp, rsrq float64,
	location model.Coordinate,
	neighborCells, interferingBeams []*model.UECell) (*model.UE, string) {

	imsi := utils.ImsiGenerator(counter)
	ueIMSI := strconv.FormatUint(uint64(imsi), 10)

	rrcState := mho.Rrcstatus_RRCSTATUS_CONNECTED
	// add neighbours
	pCell := &model.UECell{
		ID:   types.GnbID(ncgi),
		NCGI: types.NCGI(ncgi),
		BeamID: model.BeamID{
			NCGI:         beamQS.BeamID.NCGI,
			CarrierIndex: beamQS.BeamID.CarrierIndex,
			BeamIndex:    beamQS.BeamID.BeamIndex,
		},
		Rsrq:        rsrq,
		Rsrp:        rsrp,
		Sinr:        sinr,
		AvailPrbsDl: totalPrbsDl,
	}

	ue := &model.UE{
		IMSI:                      imsi,
		AmfUeNgapID:               types.AmfUENgapID(1000 + counter),
		Type:                      "phone",
		Location:                  location,
		Heading:                   0,
		ServingCells:              []*model.UECell{pCell},
		FiveQi:                    beamQS.CQI,
		CRNTI:                     types.CRNTI(90125 + counter),
		NeighborCells:             []*model.UECell{},
		InterferingBeams:          interferingBeams,
		IsAdmitted:                false,
		Height:                    ueHeight,
		RrcState:                  rrcState,
		SupportedBandCombinations: map[model.ConnectivityType]*model.ConnTypeSupportInfo{},
		SupportedBandsNR:          []string{},
		SupportedBandsEutra:       []string{},
	}

	sCells := signal.TopKCellsByRSRP(*ue, neighborCells, 1+rand.Intn(5))
	ue.ServingCells = append(ue.ServingCells, sCells...)

	for _, nCell := range neighborCells {
		if _, isServing := ue.GetServingCell(nCell.NCGI); isServing {
			continue
		}
		ue.NeighborCells = append(ue.NeighborCells, nCell)
	}

	// TODO: use assigned nCells!
	initUEConnectivity(ue, maps.Values(nCells))

	return ue, ueIMSI
}

func findInterferingBeams(point model.Coordinate, sCell *model.Cell, beamID model.BeamID, cells map[string]*model.Cell, ueHeight float64, prbMeasPerCell map[uint64]map[string]int) []*model.UECell {
	ueNeighCellNCGIs := mapset.NewSet[types.NCGI]()
	ueNeighbors := []*model.UECell{}

	interferingBeamIDs, interferingCells := signal.GetInterferingBeams(point, sCell, beamID, cells)

	for _, nBeamID := range interferingBeamIDs {
		nCell, ok := interferingCells[nBeamID.NCGI]
		if !ok || ueNeighCellNCGIs.Contains(nCell.NCGI) {
			continue
		}

		nCarrier := nCell.GetCarrier(nBeamID)
		mpf := signal.RiceanFading(signal.GetRiceanK(nCarrier))
		interfBeamIDs, interfCells := signal.GetInterferingBeams(point, nCell, nBeamID, cells)
		rsrp := signal.Strength(point, ueHeight, mpf, nCell, nBeamID)
		sinr := signal.Sinr(point, ueHeight, nCell, nBeamID, interfBeamIDs, interfCells)
		rsrq := signal.RSRQ(sinr, 24)

		ueCell := &model.UECell{
			NCGI:        nBeamID.NCGI,
			BeamID:      nBeamID,
			Rsrp:        math.Round(rsrp*100) / 100,
			Rsrq:        math.Round(rsrq*100) / 100,
			Sinr:        math.Round(sinr*100) / 100,
			AvailPrbsDl: prbMeasPerCell[uint64(nBeamID.NCGI)][bw.AVAIL_PRBS_DL_METRIC],
		}

		ueNeighbors = append(ueNeighbors, ueCell)
		ueNeighCellNCGIs.Add(nCell.NCGI)

	}

	return ueNeighbors
}

func pickRandomlyFromSlice(slice []string) string {
	if len(slice) == 1 {
		return slice[0]
	}
	return slice[rand.Intn(len(slice))]
}

func initUEConnectivity(ue *model.UE, cells []*model.Cell) {
	cas := map[model.ConnectivityType]bw.CarrierAggregator{
		model.EUTRA: bw.NewCarrierAggregatorEUTRA(),
		model.NR:    bw.NewCarrierAggregatorNR(),
	}

	bandSupportInfo := bw.GetBandSupportInfo(cells)
	log.Debugf("bandSupportInfo: \n%+v", bandSupportInfo)
	validCABandCombosByConnType := bw.GetValidCABandCombosByConnType(bandSupportInfo, cas)
	log.Debugf("validCABandCombosByConnType: \n%+v", validCABandCombosByConnType)

	anyValidBandCombo := false
	var supportedConnType model.ConnectivityType
	for ct := range validCABandCombosByConnType {
		anyValidBandCombo = len(validCABandCombosByConnType[ct]) > 0
		if anyValidBandCombo {
			supportedConnType = ct
			break
		}
	}

	if !anyValidBandCombo {
		log.Warnf("ue: %v | could not init CA support, no available combos found", ue.IMSI)
		return
	}

	ue.SupportedBandCombinations[supportedConnType] = &model.ConnTypeSupportInfo{
		SupportedBandCombinations: []*model.BandCombination{},
	}

	numCombos := 1 + rand.Intn(4)
	addedCombos := mapset.NewSet[string]()
	for c := 0; c < numCombos; c++ {
		combo := pickRandomlyFromSlice(validCABandCombosByConnType[supportedConnType])
		if !addedCombos.Add(combo) {
			continue
		}
		comboBands := strings.Split(combo, "_")
		bandSupportInfo := []*model.BandSupportInfo{}

		for _, band := range comboBands {
			bandSupportInfo = append(bandSupportInfo, &model.BandSupportInfo{
				Band:           band,
				BandwidthClass: pickRandomlyFromSlice(bw.BandwidthClassesNR),
				MIMOLayers:     pickRandomlyFromSlice([]string{"2", "4", "8", "16", "32"}),
			})
		}

		ue.SupportedBandCombinations[supportedConnType].SupportedBandCombinations = append(
			ue.SupportedBandCombinations[supportedConnType].SupportedBandCombinations,
			&model.BandCombination{
				CombinedBandsInfo: bandSupportInfo,
			},
		)

		switch supportedConnType {
		case model.EUTRA:
			ue.SupportedBandsEutra = append(ue.SupportedBandsEutra, comboBands...)
		case model.NR:
			ue.SupportedBandsNR = append(ue.SupportedBandsNR, comboBands...)
		}
	}

}
