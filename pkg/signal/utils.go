package signal

import (
	"sort"

	"github.com/nfvri/onos-api/go/onos/ransim/types"
	"github.com/nfvri/ran-simulator/pkg/model"
	"github.com/sirupsen/logrus"
)

const MIN_ACCEPTABLE_RSRP = -110.0

func TopKCellsByRSRP(ue model.UE, ueCells []*model.UECell, k int) []*model.UECell {
	bestRSRPsByNCGI := map[types.NCGI]*model.UECell{}

	for _, cell := range ueCells {
		if existingCell, exists := bestRSRPsByNCGI[cell.NCGI]; !exists || cell.Rsrp > existingCell.Rsrp {
			bestRSRPsByNCGI[cell.NCGI] = cell
		}
	}

	rankedCells := make([]*model.UECell, 0, len(bestRSRPsByNCGI))
	for _, cell := range bestRSRPsByNCGI {
		if cell.Rsrp > MIN_ACCEPTABLE_RSRP {
			rankedCells = append(rankedCells, cell)
		}
	}

	sort.Slice(rankedCells, func(i, j int) bool {
		return rankedCells[i].Rsrp > rankedCells[j].Rsrp
	})

	if len(rankedCells) > k {
		rankedCells = rankedCells[:k]
	}

	logrus.Infof("ue: %v | rankTargetCellsByRSRP -> rankedCells: %v", ue.IMSI, rankedCells)
	return rankedCells
}
