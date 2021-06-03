// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * This file is part of the IoT Agent
 * Copyright 2019 Canonical Ltd.
 *
 * This program is free software: you can redistribute it and/or modify it
 * under the terms of the GNU General Public License version 3, as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful, but WITHOUT
 * ANY WARRANTY; without even the implied warranties of MERCHANTABILITY,
 * SATISFACTORY QUALITY, or FITNESS FOR A PARTICULAR PURPOSE.
 * See the GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package snapdapi

import (
	"errors"
	"log"

	"github.com/snapcore/snapd/asserts"
)

// ActionDevice has basic information of a device
type ActionDevice struct {
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	SerialNumber string `json:"serial"`
	StoreID      string `json:"store"`
	DeviceKey    string `json:"deviceKey"`
}

// GetEncodedAssertions fetches the encoded model and serial assertions
func (a *ClientAdapter) GetEncodedAssertions(assertionType string) ([]byte, error) {
	at := asserts.Type(assertionType)
	if at == nil {
		return nil, errors.New("unknown assertion type")
	}

	if at.Name != asserts.ModelType.Name {
		return nil, errors.New("only model assertions are supported currently")
	}

	// Get the model assertion
	modelAssertions, err := a.Known(asserts.ModelType.Name, map[string]string{})
	if err != nil || len(modelAssertions) == 0 {
		log.Printf("error retrieving the model assertion: %v", err)
		return nil, err
	}
	dataModel := asserts.Encode(modelAssertions[0])

	data := append(dataModel, []byte("\n")...)
	return data, nil
}

// DeviceInfo fetches the basic details of the device
func (a *ClientAdapter) DeviceInfo() (ActionDevice, error) {
	// Get the model assertion
	modelAssertions, err := a.Known(asserts.ModelType.Name, map[string]string{})
	if err != nil || len(modelAssertions) == 0 {
		log.Printf("error retrieving the model assertion: %v", err)
		return ActionDevice{}, err
	}
	model := modelAssertions[0]

	// Get the serial assertion
	serialAssertions, err := a.Known(asserts.SerialType.Name, map[string]string{})
	if err != nil || len(serialAssertions) == 0 {
		log.Printf("error retrieving the serial assertion: %v", err)
		return ActionDevice{}, err
	}
	serial := serialAssertions[0]

	return ActionDevice{
		Brand:        serial.HeaderString("brand-id"),
		Model:        serial.HeaderString("model"),
		SerialNumber: serial.HeaderString("serial"),
		DeviceKey:    serial.HeaderString("device-key"),
		StoreID:      model.HeaderString("store"),
	}, nil
}
