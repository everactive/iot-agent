package server

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/everactive/iot-identity/domain"
	"github.com/sirupsen/logrus"

	"github.com/everactive/iot-agent/config"
	"github.com/everactive/iot-agent/identity"
	"github.com/everactive/iot-agent/mqtt"
	"github.com/everactive/iot-agent/pkg/legacy"
	"github.com/everactive/iot-agent/pkg/nats"
	"github.com/everactive/iot-agent/snapdapi"
)

var tickInterval = 60
var enrollTickInterval = 2

var createNATSServer = func() AddOnServer {
	server := nats.NewServer()
	return server
}

type AddOnServer interface {
	Start() error
	Stop() error
	SetLegacy(*legacy.HandlerIFace)
}

type Server struct {
	settings           *config.Settings
	snapdClientAdapter *snapdapi.ClientAdapter
	legacy             legacy.HandlerIFace
	legacyLock         sync.Mutex
	serversLock        sync.Mutex
	runningLock        sync.Mutex
	isRunning          bool
	otherServers       []AddOnServer
	identity           identity.Identity
}

var Clock clock.Clock

func New() *Server {
	Clock = clock.New()

	// Set up the dependency chain
	settings := config.ReadParameters()
	snap := snapdapi.NewClientAdapter()

	// Check that we are enrolled with the identity service
	idSrv := identity.NewService(settings, snap)

	return &Server{
		settings:           settings,
		snapdClientAdapter: snap,
		otherServers:       []AddOnServer{},
		identity:           idSrv,
	}
}

func (s *Server) IsRunning() bool {
	s.runningLock.Lock()
	defer s.runningLock.Unlock()
	return s.isRunning
}

func (s *Server) Run() {
	s.runningLock.Lock()
	s.isRunning = true
	s.runningLock.Unlock()

	// Server NATS responses whether we are enrolled or not
	natsServer := createNATSServer() // nats.NewServer()
	err := natsServer.Start()
	if err != nil {
		logrus.Error(err)
	}
	s.AddServer(natsServer)

	d := time.Duration(enrollTickInterval) * time.Second
	s.serversLock.Lock()
	enrollTicker := Clock.Ticker(d)
	s.serversLock.Unlock()

	for range enrollTicker.C {
		errEnroll := s.Enroll()
		if errEnroll == nil {
			enrollTicker.Stop()
			break
		}
	}

	natsServer.SetLegacy(&s.legacy)

	defer s.Stop()

	quitSignals := make(chan os.Signal, 1)
	signal.Notify(quitSignals, syscall.SIGINT, syscall.SIGTERM)

	// On an interval...
	s.serversLock.Lock()
	serviceTicker := Clock.Ticker(time.Second * time.Duration(tickInterval))
	s.serversLock.Unlock()
	go func() {
		for range serviceTicker.C {
			s.Service()
		}
		serviceTicker.Stop()
	}()

	// This will block until SIGINT or SIGTERM
	sig := <-quitSignals

	serviceTicker.Stop()

	// SIGINT is expected from systemd and should not result in an error exit
	if sig == syscall.SIGINT {
		// We expect this signal, it is not an error
		logrus.Infof("Exited because of signal %+v", sig)
		return
	}

	fmt.Printf("caught signal %v\n", sig)
}

var createLegacySubscriberVar = createLegacySubscriber

func createLegacySubscriber(enrollment *domain.Enrollment) (legacy.HandlerIFace, error) {
	// Create/get the MQTT connection
	mqttConn, err := mqtt.GetConnection(enrollment)
	if err != nil {
		log.Printf("Error with MQTT connection: %v", err)
		return nil, err
	}

	legacy := legacy.New(mqttConn, enrollment)

	// Subscribe to the actions topic
	err = legacy.SubscribeToActions()
	if err != nil {
		log.Println(err.Error())
		return legacy, err
	}

	return legacy, nil
}

func (s *Server) Enroll() error {
	enroll, err := s.identity.CheckEnrollment()

	if err != nil {
		log.Printf("Error with enrollment: %v", err)
		return err
	}

	legacy, err := createLegacySubscriberVar(enroll)
	s.legacyLock.Lock()
	defer s.legacyLock.Unlock()
	s.legacy = legacy
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) Stop() {
	s.serversLock.Lock()
	defer s.serversLock.Unlock()
	for _, srv := range s.otherServers {
		_ = srv.Stop()
	}
	s.otherServers = []AddOnServer{}

	s.legacyLock.Lock()
	defer s.legacyLock.Unlock()
	if s.legacy != nil {
		s.legacy.Close()
	}
}

func (s *Server) Service() {
	// Publish the health check and metrics messages
	s.legacy.Health()
	s.legacy.Metrics()
}

func (s *Server) AddServer(server AddOnServer) {
	s.serversLock.Lock()
	defer s.serversLock.Unlock()
	err := server.Start()
	if err != nil {
		logrus.Error(err)
	}
	s.otherServers = append(s.otherServers, server)
}
