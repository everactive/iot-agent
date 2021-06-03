// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * This file is part of the IoT Agent
 * Copyright 2020 Canonical Ltd.
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

package legacy

import (
	"fmt"
	"log"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"

	"github.com/everactive/iot-agent/mqtt"
)

// Metrics publishes a metrics messages to indicate so the device can be monitored
func (h *Handler) Metrics() {
	// Publish the stats
	h.memory()
	h.cpu()
}

func (h *Handler) publishMetrics(payload string) {
	// The topic to publish the response to the specific action
	t := fmt.Sprintf("metrics/%s", h.organizationID)
	h.mqttConn.Client.Publish(t, mqtt.QOSAtMostOnce, false, []byte(payload))
}

func (h *Handler) memory() {
	v, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Error getting memory usage: %v\n", err)
		return
	}

	payload := fmt.Sprintf("memory,device=%s total=%d,used=%d,usedpc=%f", h.clientID, v.Total, v.Used, v.UsedPercent)
	h.publishMetrics(payload)
}

func (h *Handler) cpu() {
	vv, err := cpu.Times(false)
	if err != nil {
		log.Printf("Error getting cpu usage: %v\n", err)
		return
	}

	var user, system, total float64
	for _, v := range vv {
		user += v.User
		system += v.System
		total += v.Total()
	}

	payload := fmt.Sprintf("cpu,device=%s user=%f,system=%f,total=%f", h.clientID, user, system, total)
	h.publishMetrics(payload)
}
