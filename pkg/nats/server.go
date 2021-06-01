package nats

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/everactive/iot-agent/pkg/messages"

	natsgo "github.com/nats-io/nats.go"

	"github.com/everactive/iot-agent/pkg/legacy"

	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/client"
	"github.com/spf13/viper"

	"github.com/everactive/iot-agent/pkg/config"
	"github.com/everactive/iot-agent/snapdapi"
)

const (
	natsURL                 = "nats://localhost:4222"
	natsClientName          = "iot-agent-snapd"
	snapdAPIUsername        = "snapd"
	snapdAPIDefaultPassword = "accept solve carbon atmosphere"
)

var createQuitSignalChannel = createOSQuitSignalChannel
var snapd snapdapi.SnapdClient = snapdapi.NewClientAdapter()

func createOSQuitSignalChannel() chan os.Signal {
	quitSignals := make(chan os.Signal, 1)
	signal.Notify(quitSignals, syscall.SIGINT, syscall.SIGTERM)

	return quitSignals
}

type natsConnInterface interface {
	Publish(subject string, v interface{}) error
	Subscribe(subject string, cb natsgo.Handler) (*natsgo.Subscription, error)
}

type Server struct {
	encodedConn     natsConnInterface
	LegacyInterface *legacy.HandlerIFace
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) SetLegacy(face *legacy.HandlerIFace) {
	s.LegacyInterface = face
}

func (s *Server) Start() error {
	// Will stay here forever or until a NATS connection is available or we get a quit signal
	nc := s.getInitialNATSConnection(natsURL, natsClientName)
	if nc == nil {
		logrus.Error("NATS connection is nil. Exiting.")
		return errors.New("NATS connection is nil")
	}

	c, err := natsgo.NewEncodedConn(nc, natsgo.JSON_ENCODER)
	if err != nil {
		logrus.Fatal("error starting encoded connection: ", err)
	}

	s.encodedConn = c

	s.setupSubscriptions()

	return nil
}

func (s *Server) Stop() error {
	return nil
}

func (s *Server) getInitialNATSConnection(natsURL, clientName string) *natsgo.Conn {
	quitSignals := createQuitSignalChannel()

	opts := []natsgo.Option{natsgo.Name(clientName)}
	var password string
	if viper.IsSet(config.NATSSnapdPasswordKey) {
		password = viper.GetString(config.NATSSnapdPasswordKey)
	} else {
		password = snapdAPIDefaultPassword
	}
	opts = append(opts, natsgo.UserInfo(snapdAPIUsername, password))

	logrus.Infof("Trying to connecting to NATS server at %s", natsURL)
	nc, err := natsgo.Connect(natsURL, opts...)
	if err != nil {
		logrus.Errorf("error connecting to NATS: %s, will retry.", err)

		duration := viper.GetDuration(config.NATSConnectionRetryIntervalKey)
		timer := time.NewTicker(duration)

		done := false
		var errInternal error
		for !done {
			select {
			case <-timer.C:
				nc, errInternal = natsgo.Connect(natsURL, opts...)
				if errInternal != nil {
					logrus.Errorf("error connecting to NATS: %s, will retry.", errInternal)
				} else {
					timer.Stop()
					done = true
				}
			case <-quitSignals:
				timer.Stop()
				logrus.Warnf("Quit signal received, quitting.")
				return nil
			}
		}
	}

	logrus.Infof("Connected to NATS server at %s", natsURL)
	return nc
}

func (s *Server) handleAssertionsGet(_ string, reply string, message AssertionsRequest) {
	fmt.Printf("%+v\n", message)

	assertions, err := snapd.GetEncodedAssertions(message.Type)
	if err != nil {
		logrus.Error(err)
		return
	}

	response := &AssertionsResponse{Stream: string(assertions)}

	err = s.encodedConn.Publish(reply, response)
	if err != nil {
		logrus.Error(err)
	}
}

func (s *Server) handleAssertionsGetv1(_ string, reply string, message messages.AssertionsRequest) {
	fmt.Printf("%+v\n", message)

	assertions, err := snapd.GetEncodedAssertions(message.Type)
	if err != nil {
		logrus.Error(err)
		return
	}

	response := &messages.AssertionsResponse{Stream: string(assertions)}

	err = s.encodedConn.Publish(reply, response)
	if err != nil {
		logrus.Error(err)
	}
}

func (s *Server) handleSnapsGet(_ string, reply string, _ emptyMessage) {
	snaps, err := snapd.Snaps()
	if err != nil {
		logrus.Error(err)
		return
	}

	response := &messages.SnapsResponse{}
	for _, sn := range snaps {
		snap := makeSnapsResponseSnap(sn)
		response.Snaps = append(response.Snaps, &snap)
	}

	err = s.encodedConn.Publish(reply, response)
	if err != nil {
		logrus.Error(err)
	}
}

func (s *Server) handleSnapsGetv1(_ string, reply string, _ emptyMessage) {
	snaps, err := snapd.Snaps()
	if err != nil {
		logrus.Error(err)
		return
	}

	response := &messages.SnapsResponse{}
	for _, sn := range snaps {
		snap := makeSnapsResponseSnap(sn)
		response.Snaps = append(response.Snaps, &snap)
	}

	err = s.encodedConn.Publish(reply, response)
	if err != nil {
		logrus.Error(err)
	}
}

func (s *Server) handleAppsPost(_ string, reply string, message *AppsRequest) {
	if message.Action != "stop" && message.Action != "start" {
		logrus.Error("Only start and stop actions are supported")
		return
	}

	var changeID string
	var err error
	if message.Action == "start" {
		changeID, err = snapd.Start(message.Names, client.StartOptions{})
	} else {
		changeID, err = snapd.Stop(message.Names, client.StopOptions{})
	}

	if err != nil {
		logrus.Error(err)
		errorMessage := err.Error()
		response := AppsResponse{
			ChangeID: changeID,
			Error:    &errorMessage,
		}

		err = s.encodedConn.Publish(reply, response)
		if err != nil {
			logrus.Error(err)
		}

		return
	}

	logrus.Infof("Action=%s, for %+v started. Change id %s", message.Action, message.Names, changeID)

	response := AppsResponse{
		ChangeID: changeID,
	}

	err = s.encodedConn.Publish(reply, response)
	if err != nil {
		logrus.Error(err)
	}
}

func (s *Server) handleAppsPostv1(_ string, reply string, message *messages.AppsRequest) {
	if message.Action != "stop" && message.Action != "start" {
		logrus.Error("Only start and stop actions are supported")
		return
	}

	var changeID string
	var err error
	if message.Action == "start" {
		changeID, err = snapd.Start(message.Names, client.StartOptions{})
	} else {
		changeID, err = snapd.Stop(message.Names, client.StopOptions{})
	}

	if err != nil {
		logrus.Error(err)
		errorMessage := err.Error()
		response := messages.AppsResponse{
			ChangeId: changeID,
			Error:    errorMessage,
		}

		err = s.encodedConn.Publish(reply, response)
		if err != nil {
			logrus.Error(err)
		}

		return
	}

	logrus.Infof("Action=%s, for %+v started. Change id %s", message.Action, message.Names, changeID)

	response := messages.AppsResponse{
		ChangeId: changeID,
	}

	err = s.encodedConn.Publish(reply, response)
	if err != nil {
		logrus.Error(err)
	}
}

func (s *Server) handleMqttConnectionStatus(_ string, reply string, message *messages.MqttConnectionStatusRequest) {
	response := &messages.MqttConnectionStatus{
		ErrorInfo: nil,
		Connected: false,
	}

	if nil == s.LegacyInterface {
		response.ErrorInfo = &messages.ErrorInfo{
			Message: "Mqtt connection status request but no LegacyInterface has not been set. Perhaps this request happened before iot-agent has finished starting up",
		}
	} else {
		response.Connected = (*s.LegacyInterface).IsConnected()
	}

	err := s.encodedConn.Publish(reply, response)
	if err != nil {
		logrus.Error(err)
	}
}

func (s *Server) handleSnapsSnapPostv1(subject string, reply string, message *messages.SnapsSnapRequest) {
	response := messages.AsyncResponse{
		ChangeId: "-1",
		Error:    "Unknown error",
	}

	defer func() {
		err := s.encodedConn.Publish(reply, response)
		if err != nil {
			logrus.Error(err)
		}
	}()

	if message.Action != "switch" {
		logrus.Error("Only the switch action is supported")
		response.Error = "Invalid action"
		return
	}

	subjectParts := strings.Split(subject, ".")
	if len(subjectParts) != 6 {
		logrus.Error("subject is unexpected length")
		response.Error = "Subject is unexpected length"
		return
	}

	snapName := subjectParts[4]

	changeID, err := snapd.Switch(snapName, &client.SnapOptions{Channel: message.Channel})

	if err != nil {
		logrus.Error(err)
		response.ChangeId = changeID
		response.Error = err.Error()
		return
	}

	logrus.Infof("Action=%s, for %+v started. Change id %s", message.Action, snapName, changeID)

	response.Error = ""
	response.ChangeId = changeID
}

func (s *Server) setupSubscriptions() {
	// maps string -> server handler function
	subscriptions := map[string]interface{}{
		AssertionGetSubject:         s.handleAssertionsGet,
		SnapsGetSubject:             s.handleSnapsGet,
		AppsPostSubject:             s.handleAppsPost,
		AssertionGetSubjectv1:       s.handleAssertionsGetv1,
		SnapsGetSubjectv1:           s.handleSnapsGetv1,
		AppsPostSubjectv1:           s.handleAppsPostv1,
		SnapsSnapPostSubjectv1:      s.handleSnapsSnapPostv1,
		IotAgentMqttBrokerConnected: s.handleMqttConnectionStatus,
	}

	for subject, handlerFn := range subscriptions {
		_, err := s.encodedConn.Subscribe(subject, handlerFn)
		if err != nil {
			logrus.Errorf("Error with subject %s", subject)
			logrus.Error(err)
		} else {
			logrus.Infof("Subscribed to subject: %s", subject)
		}
	}
}

func makeSnapsResponseApps(snap *client.Snap) []*messages.AppInfo {
	var apps []*messages.AppInfo
	for _, app := range snap.Apps {
		apps = append(apps, &messages.AppInfo{
			Active:  app.Active,
			Daemon:  app.Daemon,
			Enabled: app.Enabled,
			Name:    app.Name,
		})
	}
	return apps
}

func makeSnapsResponseSnap(snap *client.Snap) messages.Snap {
	apps := makeSnapsResponseApps(snap)
	s := messages.Snap{
		Apps:            apps,
		Channel:         snap.Channel,
		Id:              snap.ID,
		InstallDate:     snap.InstallDate,
		InstalledSize:   snap.InstalledSize,
		Name:            snap.Name,
		Revision:        snap.Revision.String(),
		Status:          snap.Status,
		Title:           snap.Title,
		TrackingChannel: snap.TrackingChannel,
		Type:            snap.Type,
		Version:         snap.Version,
	}

	return s
}
