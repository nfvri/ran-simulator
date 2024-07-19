package signal

import (
	"math"
	"math/rand"
	"strconv"

	mho "github.com/onosproject/onos-e2-sm/servicemodels/e2sm_mho_go/v2/e2sm-mho-go"

	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/nfvri/ran-simulator/pkg/store/ues"
	"github.com/onosproject/onos-api/go/onos/ransim/types"
	log "github.com/sirupsen/logrus"
)

// CQItoSINR mapping
// 0 and 16 values included only for calculations, not valid CQI indexes
var CQItoSINRmap = map[int]float64{
	0:  -8.950,
	1:  -6.9360,
	2:  -5.1470,
	3:  -3.1800,
	4:  -1.2530,
	5:  0.7610,
	6:  2.6990,
	7:  4.6940,
	8:  6.5250,
	9:  8.5730,
	10: 10.3660,
	11: 12.2890,
	12: 14.1730,
	13: 15.8880,
	14: 17.8140,
	15: 19.8290,
	16: 21.843,
}

func GetSINR(cqi int) float64 {

	lowerBound := CQItoSINRmap[cqi-1]
	upperBound := CQItoSINRmap[cqi]

	sinr := lowerBound + math.Abs(rand.Float64()*(upperBound-lowerBound))
	log.Infof("CQI: %d -- sinr: %f", cqi, sinr)
	return sinr
}

func CreateSimulationUE(ncgi uint64, counter int, sinr float64, location *model.Coordinate) (*model.UE, string) {

	imsi := types.IMSI(rand.Int63n(ues.MaxIMSI-ues.MinIMSI) + ues.MinIMSI)
	ueIMSI := strconv.FormatUint(uint64(imsi), 10)

	rrcState := mho.Rrcstatus_RRCSTATUS_CONNECTED

	servingCell := &model.UECell{
		ID:   types.GnbID(ncgi),
		NCGI: types.NCGI(ncgi),
		Rsrp: rand.Float64() * 100,
		Sinr: sinr,
	}

	ue := &model.UE{
		IMSI:        imsi,
		AmfUeNgapID: types.AmfUENgapID(1000 + counter),
		Type:        "phone",
		Location:    *location,
		Heading:     0,
		Cell:        servingCell,
		CRNTI:       types.CRNTI(90125 + counter),
		Cells:       []*model.UECell{servingCell},
		IsAdmitted:  false,
		RrcState:    rrcState,
	}

	return ue, ueIMSI
}

func CalculateUEsLocations(ncgi uint64, numUes int, sinr float64, simModel *model.Model) ([]*model.Coordinate, error) {

	cell := getCell(ncgi, simModel)
	cellCenter := cell.Sector.Center
	log.Infof("NCGI: %d -- Center: %v", ncgi, cellCenter)
	ueLocations := []*model.Coordinate{}
	for i := 0; i < numUes; i++ {
		coord := &model.Coordinate{Lat: (cellCenter.Lat + float64(i)/100), Lng: (cellCenter.Lng + float64(i)/100)}
		ueLocations = append(ueLocations, coord)
	}
	log.Info("ueLocations: ")
	for i := 0; i < numUes; i++ {
		log.Infof("%v", ueLocations[i])
	}

	return ueLocations, nil
}

func getCell(ncgi uint64, simModel *model.Model) *model.Cell {

	NCGI := types.NCGI(ncgi)
	var foundCell *model.Cell

	for _, cell := range simModel.Cells {
		if cell.NCGI == NCGI {
			foundCell = &cell
			break
		}
	}
	return foundCell
}
