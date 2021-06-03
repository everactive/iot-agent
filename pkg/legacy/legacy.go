package legacy

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

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"

	"os"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/everactive/iot-devicetwin/pkg/actions"
	"github.com/everactive/iot-devicetwin/pkg/messages"
	identity "github.com/everactive/iot-identity/domain"

	"github.com/everactive/iot-agent/mqtt"
	"github.com/everactive/iot-agent/snapdapi"
)

const (
	quiesce = 250
)

var snapd snapdapi.SnapdClient = snapdapi.NewClientAdapter()

func MockSnapdClient(mockedSnapd snapdapi.SnapdClient) {
	snapd = mockedSnapd
}

type HandlerIFace interface {
	SubscribeToActions() error
	Health()
	Metrics()
	Close()
	IsConnected() bool
}

type Handler struct {
	mqttConn       *mqtt.Connection
	clientID       string
	organizationID string
	enrollment     *identity.Enrollment
	snapdClient    snapdapi.SnapdClient
}

func New(mqttConn *mqtt.Connection, enrollment *identity.Enrollment) *Handler {
	return &Handler{
		mqttConn,
		enrollment.ID,
		enrollment.Organization.ID,
		enrollment,
		snapdapi.NewClientAdapter(),
	}
}

// SubscribeToActions subscribes to the action topic
func (h *Handler) SubscribeToActions() error {
	t := fmt.Sprintf("devices/sub/%s", h.clientID)
	token := h.mqttConn.Client.Subscribe(t, mqtt.QOSAtLeastOnce, h.subscribeHandler)
	token.Wait()
	if token.Error() != nil {
		log.Printf("Error subscribing to topic `%s`: %v", t, token.Error())
		return fmt.Errorf("error subscribing to topic `%s`: %v", t, token.Error())
	}
	return nil
}

// SubscribeHandler is the handler for the main subscription topic
func (h *Handler) subscribeHandler(client MQTT.Client, msg MQTT.Message) {
	s, err := deserializePayload(msg)
	if err != nil {
		return
	}

	// The topic to publish the response to the specific action
	t := fmt.Sprintf("devices/pub/%s", h.clientID)

	// Perform the action
	response, err := h.performAction(s)
	if err != nil {
		log.Printf("Error with action `%s`: %v", s.Action, err)
	}

	// Publish the response to the action to the broker
	client.Publish(t, mqtt.QOSAtLeastOnce, false, response)

	// Handle the special case that this action was an unregister.
	// This lives here, so that the response can be sent to the broker before
	// the process exits
	if s.Action == "unregister" && err == nil {
		log.Printf("Exiting as a result of an unregister action")
		os.Exit(0)
	}
}

// performAction acts on the topic and returns a response to publish back
func (h *Handler) performAction(s *SubscribeAction) ([]byte, error) {
	// Act based on the message action
	switch s.Action {
	case actions.Device:
		result := s.Device(h.organizationID, h.clientID)
		result.Action = s.Action
		return serializeResponse(result)
	case actions.List:
		result := s.SnapList(h.clientID)
		result.Action = s.Action
		return serializeResponse(result)
	case actions.Install:
		result := s.SnapInstall()
		result.Action = s.Action
		return serializeResponse(result)
	case actions.Remove:
		result := s.SnapRemove()
		result.Action = s.Action
		return serializeResponse(result)
	case actions.Refresh:
		result := s.SnapRefresh()
		result.Action = s.Action
		return serializeResponse(result)
	case actions.Revert:
		result := s.SnapRevert()
		result.Action = s.Action
		return serializeResponse(result)
	case actions.Enable:
		result := s.SnapEnable()
		result.Action = s.Action
		return serializeResponse(result)
	case actions.Disable:
		result := s.SnapDisable()
		result.Action = s.Action
		return serializeResponse(result)
	case actions.Start:
		result := s.SnapStart()
		result.Action = s.Action
		return serializeResponse(result)
	case actions.Stop:
		result := s.SnapStop()
		result.Action = s.Action
		return serializeResponse(result)
	case actions.Restart:
		result := s.SnapRestart()
		result.Action = s.Action
		return serializeResponse(result)
	case actions.Conf:
		result := s.SnapConf()
		result.Action = s.Action
		return serializeResponse(result)
	case actions.SetConf:
		result := s.SnapSetConf()
		result.Action = s.Action
		return serializeResponse(result)
	case actions.Info:
		result := s.SnapInfo()
		result.Action = s.Action
		return serializeResponse(result)
	case actions.Ack:
		result := s.SnapAck()
		result.Action = s.Action
		return serializeResponse(result)
	case actions.Server:
		result := s.SnapServerVersion(h.clientID)
		result.Action = s.Action
		return serializeResponse(result)
	case actions.Switch:
		result := s.SnapSwitch()
		result.Action = s.Action
		return serializeResponse(result)
	case actions.Unregister:
		result := s.Unregister(h.organizationID, h.clientID)
		result.Action = s.Action
		return serializeResponse(result)
	case actions.Logs:
		result := s.RetrieveLogs(h.snapdClient)
		result.Action = s.Action
		return serializeResponse(result)
	case actions.Snapshot:
		result := s.SnapSnapshot(h.snapdClient)
		result.Action = s.Action
		return serializeResponse(result)
	default:
		return nil, fmt.Errorf("unhandled action: %s", s.Action)
	}
}

// Health publishes a health message to indicate that the device is still active
func (h *Handler) Health() {
	// Serialize the device health details
	health := messages.Health{
		OrgId:    h.organizationID,
		DeviceId: h.clientID,
		Refresh:  time.Now(),
	}

	data, err := json.Marshal(&health)
	if err != nil {
		log.Printf("Error serializing the health data: %v", err)
		return
	}

	// The topic to publish the response to the specific action
	t := fmt.Sprintf("devices/health/%s", h.clientID)
	h.mqttConn.Client.Publish(t, mqtt.QOSAtMostOnce, false, data)
}

// Close closes the connection to the MQTT broker
func (h *Handler) Close() {
	if h.mqttConn != nil {
		h.mqttConn.Client.Disconnect(quiesce)
	}
}

func serializeResponse(resp interface{}) ([]byte, error) {
	return json.Marshal(resp)
}

func deserializePayload(msg MQTT.Message) (*SubscribeAction, error) {
	s := SubscribeAction{}

	log.Tracef("Message: %s", msg.Payload())

	// Decode the message payload - the list of snaps
	err := json.Unmarshal(msg.Payload(), &s)
	if err != nil {
		log.Println("Error decoding the subscribed message:", err)
	}
	return &s, err
}

func (h *Handler) IsConnected() bool {
	return h.mqttConn.Client.IsConnectionOpen()
}
