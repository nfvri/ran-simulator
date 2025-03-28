// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"testing"

	"github.com/nfvri/onos-api/go/onos/ransim/types"
	"github.com/stretchr/testify/assert"
)

func TestModel(t *testing.T) {
	model := &Model{}
	err := LoadConfig(model, "test")
	assert.NoError(t, err)
	t.Log(model)
	assert.Equal(t, 0, len(model.Controllers))
	assert.Equal(t, 3, len(model.Nodes))
	assert.Equal(t, 3, len(model.Cells))
	assert.Equal(t, 0, len(model.ServiceModels))
	assert.Equal(t, uint(12), model.UECount)
	assert.Equal(t, "00101", model.Plmn)
	assert.Equal(t, types.PlmnID(0x101), model.PlmnID)

	assert.Equal(t, types.NCGI(17680452386819), model.Cells["managedelement=1193046,gnbdufunction=3,nrcelldu=3"].NCGI)
	assert.Equal(t, 1, len(model.Nodes["managedelement=1193046"].Cells))
	assert.Equal(t, 37.981629, model.Cells["managedelement=1193048,gnbdufunction=1,nrcelldu=1"].Carriers[0].Center.Lat)

	assert.Equal(t, false, model.MapLayout.FadeMap)
	assert.Equal(t, 0.0, model.MapLayout.Center.Lat)
}
