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

package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/everactive/iot-identity/domain"
	"log"
)

// Constants for connecting to the MQTT broker
const (
	QOSAtMostOnce  = byte(0)
	QOSAtLeastOnce = byte(1)
	//QOSExactlyOnce = byte(2)
)

// Connection for MQTT protocol
type Connection struct {
	Client         MQTT.Client
	clientID       string
	organisationID string
}

var conn *Connection
var client MQTT.Client

// GetConnection fetches or creates an MQTT connection
func GetConnection(enroll *domain.Enrollment) (*Connection, error) {
	if conn == nil {
		// Create the client
		client, err := newClient(enroll)
		if err != nil {
			return nil, err
		}

		// Create a new connection
		conn = &Connection{
			Client:         client,
			clientID:       enroll.ID,
			organisationID: enroll.Organization.ID,
		}
	}

	// Check that we have a live connection
	if conn.Client.IsConnectionOpen() {
		return conn, nil
	}

	// Connect to the MQTT broker
	if token := conn.Client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return conn, nil
}

// newClient creates a new MQTT client
func newClient(enroll *domain.Enrollment) (MQTT.Client, error) {
	// Return the active client, if we have one
	if client != nil {
		return client, nil
	}

	// Generate a new MQTT client
	url := fmt.Sprintf("ssl://%s:%s", enroll.Credentials.MQTTURL, enroll.Credentials.MQTTPort)
	log.Println("Connect to the MQTT broker", url)

	// Generate the TLS config from the enrollment credentials
	tlsConfig, err := newTLSConfig(enroll)
	if err != nil {
		return nil, err
	}

	// Set up the MQTT client options
	opts := MQTT.NewClientOptions()
	opts.AddBroker(url)
	opts.SetClientID(enroll.ID)
	opts.SetTLSConfig(tlsConfig)
	opts.AutoReconnect = true
	//opts.SetOnConnectHandler(connectHandler)

	client = MQTT.NewClient(opts)
	return client, nil
}

// newTLSConfig sets up the certificates from the enrollment record
func newTLSConfig(enroll *domain.Enrollment) (*tls.Config, error) {
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(enroll.Organization.RootCert)

	// Import client certificate/key pair
	cert, err := tls.X509KeyPair(enroll.Credentials.Certificate, enroll.Credentials.PrivateKey)
	if err != nil {
		return nil, err
	}

	// Create tls.Config with desired TLS properties
	return &tls.Config{
		// RootCAs = certs used to verify server cert.
		RootCAs: certPool,
		// ClientAuth = whether to request cert from server.
		// Since the server is set up for SSL, this happens
		// anyways.
		ClientAuth: tls.NoClientCert,
		// ClientCAs = certs used to validate client cert.
		ClientCAs: nil,
		// InsecureSkipVerify = verify that cert contents
		// match server. IP matches what is in cert etc.
		InsecureSkipVerify: true,
		// Certificates = list of certs client sends to server.
		Certificates: []tls.Certificate{cert},
	}, nil
}
