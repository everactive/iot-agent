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
package legacy

import (
	"encoding/json"
	"log"
	"testing"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/everactive/iot-devicetwin/pkg/messages"
	"github.com/everactive/iot-identity/domain"

	"github.com/everactive/iot-agent/mqtt"
	"github.com/everactive/iot-agent/snapdapi"
)

func TestConnection_Workflow(t *testing.T) {
	m1a := `{"id": "abc123", "action":"install", "snap":"helloworld"}`
	m1b := `{"id": "abc123", "action":"install"}`
	m1c := `{"id": "abc123", "action":"install", "snap":"invalid"}`
	m2a := `{"id": "abc123", "action":"invalid", "snap":"helloworld"}`
	m2b := `\u1000`
	m3a := `{"id": "abc123", "action":"remove", "snap":"helloworld"}`
	m3b := `{"id": "abc123", "action":"remove"}`
	m3c := `{"id": "abc123", "action":"remove", "snap":"invalid"}`
	m4a := `{"id": "abc123", "action":"refresh", "snap":"helloworld"}`
	m4b := `{"id": "abc123", "action":"refresh"}`
	m4c := `{"id": "abc123", "action":"refresh", "snap":"invalid"}`
	m5a := `{"id": "abc123", "action":"revert", "snap":"helloworld"}`
	m5b := `{"id": "abc123", "action":"revert"}`
	m5c := `{"id": "abc123", "action":"revert", "snap":"invalid"}`
	m6a := `{"id": "abc123", "action":"list"}`
	m7a := `{"id": "abc123", "action":"enable", "snap":"helloworld"}`
	m7b := `{"id": "abc123", "action":"enable"}`
	m7c := `{"id": "abc123", "action":"enable", "snap":"invalid"}`
	m8a := `{"id": "abc123", "action":"disable", "snap":"helloworld"}`
	m8b := `{"id": "abc123", "action":"disable"}`
	m8c := `{"id": "abc123", "action":"disable", "snap":"invalid"}`
	m9a := `{"id": "abc123", "action":"conf", "snap":"helloworld"}`
	m9b := `{"id": "abc123", "action":"conf"}`
	m9c := `{"id": "abc123", "action":"conf", "snap":"invalid"}`
	m10a := `{"id": "abc123", "action":"setconf", "snap":"helloworld", "data":"{\"title\": \"Hello World!\"}"}`
	m10b := `{"id": "abc123", "action":"setconf"}`
	m10c := `{"id": "abc123", "action":"setconf", "snap":"invalid", "data":"{\"title\": \"Hello World!\"}"}`
	m10d := `{"id": "abc123", "action":"setconf", "snap":"helloworld", "data":"\u1000"}`
	m11a := `{"id": "abc123", "action":"info", "snap":"helloworld"}`
	m11b := `{"id": "abc123", "action":"info"}`
	m11c := `{"id": "abc123", "action":"info", "snap":"invalid"}`
	m12a := `{"id": "abc123", "action":"ack", "data":"serialized-assertion"}`
	m12b := `{"id": "abc123", "action":"ack", "data":"invalid"}`
	m13a := `{"id": "abc123", "action":"server"}`
	m14a := `{"id": "abc123", "action":"device"}`
	m15a := `{"id": "abc123", "action":"unregister"}`
	m16a := `{"id": "abc123", "action":"switch", "snap":"helloworld", "data": "latest/stable"}`
	m16b := `{"id": "abc123", "action":"switch", "snap":"helloworld", "data": ""}`
	m16c := `{"id": "abc123", "action":"switch", "snap":"invalid", "data": ""}`

	snapStartValid := `{"id": "abc123", "action":"start", "snap":"helloworld", "data":"{}"}`
	snapStopValid := `{"id": "abc123", "action":"stop", "snap":"helloworld", "data":"{}"}`
	snapRestartValid := `{"id": "abc123", "action":"stop", "snap":"helloworld", "data":"{}"}`

	snapStartNoSnap := `{"id": "abc123", "action":"start", "snap":"", "data":"{}"}`
	snapStopNoSnap := `{"id": "abc123", "action":"stop", "snap":"", "data":"{}"}`
	snapRestartNoSnap := `{"id": "abc123", "action":"stop", "snap":"", "data":"{}"}`

	snapStartValidMany := `{"id": "abc123", "action":"start", "snap":"helloworld", "data":"{\"services\":[\"helloworld.service1\",\"helloworld.service2\"]}"}`
	snapStopValidMany := `{"id": "abc123", "action":"stop", "snap":"helloworld", "data":"{\"services\":[\"helloworld.service1\",\"helloworld.service2\"]}"}`
	snapRestartValidMany := `{"id": "abc123", "action":"stop", "snap":"helloworld", "data":"{\"services\":[\"helloworld.service1\",\"helloworld.service2\"]}"}`

	snapStartInvalidMany := `{"id": "abc123", "action":"start", "snap":"helloworld", "data":"{\"services\":[\"helloworld.service1\",\"jelloworld.service2\"]}"}`
	snapStopInvalidMany := `{"id": "abc123", "action":"stop", "snap":"helloworld", "data":"{\"services\":[\"helloworld.service1\",\"jelloworld.service2\"]}"}`
	snapRestartInvalidMany := `{"id": "abc123", "action":"stop", "snap":"helloworld", "data":"{\"services\":[\"helloworld.service1\",\"jelloworld.service2\"]}"}`

	enroll := &domain.Enrollment{
		Credentials: domain.Credentials{
			MQTTURL:  "localhost",
			MQTTPort: "8883",
		},
	}
	client := &MockClient{}

	mqttConnection := mqtt.Connection{Client: client}
	handler := New(&mqttConnection, enroll)

	tests := []struct {
		name     string
		open     bool
		message  MQTT.Message
		snapdErr bool
		withErr  bool
		respErr  bool
	}{
		{"valid-closed", false, &MockMessage{[]byte(m1a)}, false, false, false},
		{"valid-open", true, &MockMessage{[]byte(m1a)}, false, false, false},
		{"no-snap", true, &MockMessage{[]byte(m1b)}, false, false, true},
		{"invalid-install", true, &MockMessage{[]byte(m1c)}, false, false, true},

		{"invalid-action", true, &MockMessage{[]byte(m2a)}, false, true, true},
		{"bad-data", true, &MockMessage{[]byte(m2b)}, false, true, true},

		{"valid-remove", true, &MockMessage{[]byte(m3a)}, false, false, false},
		{"no-snap-remove", true, &MockMessage{[]byte(m3b)}, false, false, true},
		{"invalid-remove", true, &MockMessage{[]byte(m3c)}, false, false, true},

		{"valid-refresh", true, &MockMessage{[]byte(m4a)}, false, false, false},
		{"no-snap-refresh", true, &MockMessage{[]byte(m4b)}, false, false, true},
		{"invalid-refresh", true, &MockMessage{[]byte(m4c)}, false, false, true},

		{"valid-revert", true, &MockMessage{[]byte(m5a)}, false, false, false},
		{"no-snap-revert", true, &MockMessage{[]byte(m5b)}, false, false, true},
		{"invalid-revert", true, &MockMessage{[]byte(m5c)}, false, false, true},

		{"valid-list", true, &MockMessage{[]byte(m6a)}, false, false, false},
		{"snapd-error-list", true, &MockMessage{[]byte(m6a)}, true, false, true},

		{"valid-enable", true, &MockMessage{[]byte(m7a)}, false, false, false},
		{"no-snap-enable", true, &MockMessage{[]byte(m7b)}, false, false, true},
		{"invalid-enable", true, &MockMessage{[]byte(m7c)}, false, false, true},

		{"valid-disable", true, &MockMessage{[]byte(m8a)}, false, false, false},
		{"no-snap-disable", true, &MockMessage{[]byte(m8b)}, false, false, true},
		{"invalid-disable", true, &MockMessage{[]byte(m8c)}, false, false, true},

		{"valid-conf", true, &MockMessage{[]byte(m9a)}, false, false, false},
		{"no-snap-conf", true, &MockMessage{[]byte(m9b)}, false, false, true},
		{"invalid-conf", true, &MockMessage{[]byte(m9c)}, false, false, true},

		{"valid-setconf", true, &MockMessage{[]byte(m10a)}, false, false, false},
		{"no-snap-setconf", true, &MockMessage{[]byte(m10b)}, false, false, true},
		{"invalid-setconf", true, &MockMessage{[]byte(m10c)}, false, false, true},
		{"bad-data-setconf", true, &MockMessage{[]byte(m10d)}, false, false, true},

		{"valid-info", false, &MockMessage{[]byte(m11a)}, false, false, false},
		{"no-snap-info", true, &MockMessage{[]byte(m11b)}, false, false, true},
		{"invalid-info", true, &MockMessage{[]byte(m11c)}, false, false, true},

		{"valid-ack", false, &MockMessage{[]byte(m12a)}, false, false, false},
		{"invalid-ack", false, &MockMessage{[]byte(m12b)}, false, false, true},

		{"valid-server", false, &MockMessage{[]byte(m13a)}, false, false, false},
		{"snapd-error-server", false, &MockMessage{[]byte(m13a)}, true, false, true},

		{"valid-deviceinfo", false, &MockMessage{[]byte(m14a)}, false, false, false},
		{"snapd-error-deviceinfo", false, &MockMessage{[]byte(m14a)}, true, false, true},

		{"valid-start", true, &MockMessage{[]byte(snapStartValid)}, false, false, false},
		{"valid-stop", true, &MockMessage{[]byte(snapStopValid)}, false, false, false},
		{"valid-restart", true, &MockMessage{[]byte(snapRestartValid)}, false, false, false},

		{"invalid-start", true, &MockMessage{[]byte(snapStartNoSnap)}, false, false, true},
		{"invalid-stop", true, &MockMessage{[]byte(snapStopNoSnap)}, false, false, true},
		{"invalid-restart", true, &MockMessage{[]byte(snapRestartNoSnap)}, false, false, true},

		{"valid-start-many", true, &MockMessage{[]byte(snapStartValidMany)}, false, false, false},
		{"valid-stop-many", true, &MockMessage{[]byte(snapStopValidMany)}, false, false, false},
		{"valid-restart-many", true, &MockMessage{[]byte(snapRestartValidMany)}, false, false, false},

		{"invalid-start-many", true, &MockMessage{[]byte(snapStartInvalidMany)}, false, false, true},
		{"invalid-stop-many", true, &MockMessage{[]byte(snapStopInvalidMany)}, false, false, true},
		{"invalid-restart-many", true, &MockMessage{[]byte(snapRestartInvalidMany)}, false, false, true},

		{"valid-unregister", false, &MockMessage{[]byte(m15a)}, false, false, false},
		{"snapd-error-unregister", false, &MockMessage{[]byte(m15a)}, true, false, true},

		{"valid-switch", true, &MockMessage{[]byte(m16a)}, false, false, false},
		{"no-channel-switch", true, &MockMessage{[]byte(m16b)}, false, false, true},
		{"invalid-switch", true, &MockMessage{[]byte(m16c)}, false, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MockSnapdClient(&snapdapi.MockClient{WithError: tt.snapdErr})

			// Check again with the action
			sa, err := deserializePayload(tt.message)
			if err != nil && !tt.withErr {
				t.Error("TestConnection_Workflow: payload - expected error got none")
				return
			}
			resp, err := handler.performAction(sa)
			if err != nil && !tt.withErr {
				t.Error("TestConnection_Workflow: action - expected error got none")
				return
			}

			r, err := deserializePublishResponse(resp)
			if err != nil && !tt.withErr {
				t.Errorf("TestConnection_Workflow: publish response: %v", err)
				return
			}
			if r == nil {
				t.Error("TestConnection_Workflow: publish response is nil")
				return
			}
			if r.Success == tt.respErr {
				t.Errorf("TestConnection_Workflow: publish response unexpected: %s", r.Message)
			}
		})
	}
}

func deserializePublishResponse(data []byte) (*messages.PublishResponse, error) {
	s := messages.PublishResponse{}

	// Decode the message payload - the list of snaps
	err := json.Unmarshal(data, &s)
	if err != nil {
		log.Println("Error decoding the published message:", err)
	}
	return &s, err
}
