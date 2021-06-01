package server

import (
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/benbjohnson/clock"
	"github.com/everactive/iot-identity/domain"
	"github.com/snapcore/snapd/osutil"
	"github.com/stretchr/testify/suite"

	"github.com/everactive/iot-agent/mocks"

	"github.com/everactive/iot-agent/config"
	"github.com/everactive/iot-agent/pkg/legacy"
)

type ServerTestSuite struct {
	suite.Suite
	serverLock sync.Mutex
	srv        *Server
}

var mockedClock *clock.Mock

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

func (s *ServerTestSuite) SetupTest() {
	if osutil.FileExists(config.GetPath("params")) {
		os.Remove(config.GetPath("params"))
	}

	createNATSServer = func() AddOnServer {
		m := mocks.AddOnServer{}
		m.On("Start").Return(nil)
		m.On("Stop").Return(nil)
		m.On("SetLegacy", mock.Anything).Return()
		return &m
	}

	s.srv = New()

	s.serverLock.Lock()
	defer s.serverLock.Unlock()
	mockedClock = clock.NewMock()
	Clock = mockedClock
}

func (s *ServerTestSuite) TearDownTest() {
	os.Remove("./params")
}

func (s *ServerTestSuite) Test_NewServer() {
	s.Assert().NotNil(s.srv)
	s.Assert().FileExists(config.GetPath("params"))
}

func (s *ServerTestSuite) Test_NewServerRun() {
	s.serverLock.Lock()
	go func() {
		s.srv.Run()
	}()

	for s.srv.IsRunning() == false {
		time.Sleep(time.Second)
	}

	s.srv.Stop()

	s.Assert().NotNil(s.srv)
	s.Assert().FileExists(config.GetPath("params"))
	s.serverLock.Unlock()
}

func (s *ServerTestSuite) Test_NewServerRun_WithEnroll() {
	m := mocks.AddOnServer{}
	createNATSServer = func() AddOnServer {
		m.On("Start").Return(nil)
		m.On("Stop").Return(nil)
		m.On("SetLegacy", mock.Anything).Return()
		return &m
	}
	s.serverLock.Lock()
	enrollment := &domain.Enrollment{
		ID:           "",
		Device:       domain.Device{},
		Credentials:  domain.Credentials{},
		Organization: domain.Organization{},
		Status:       0,
		DeviceData:   "",
	}
	mockedIdentity := &mocks.Identity{}
	mockedIdentity.On("CheckEnrollment").Return(enrollment, nil).Once()
	s.srv.identity = mockedIdentity

	tickInterval = 2

	mockedLegacy := &mocks.HandlerIFace{}
	mockedLegacy.On("Health").Return(nil).Once()
	mockedLegacy.On("Metrics").Return(nil).Once()
	mockedLegacy.On("Close").Return(nil).Once()

	createLegacySubscriberVar = func(_ *domain.Enrollment) (legacy.HandlerIFace, error) {
		return mockedLegacy, nil
	}

	go func() {
		s.srv.Run()
	}()
	runtime.Gosched()

	// Give enough time for enrollment and one round of periodic messages to happen
	sleepTime := tickInterval + enrollTickInterval + 1
	for initial := 0; initial < sleepTime; initial++ {
		mockedClock.Add(1 * time.Second)
	}

	time.Sleep(time.Duration(sleepTime+1) * time.Second)

	s.srv.Stop()

	s.Assert().NotNil(s.srv)
	s.Assert().FileExists(config.GetPath("params"))

	mockedIdentity.AssertExpectations(s.T())
	mockedLegacy.AssertExpectations(s.T())

	m.AssertCalled(s.T(), "SetLegacy", mock.Anything)
	s.serverLock.Unlock()
}

func (s *ServerTestSuite) Test_NewServerStop() {
	s.serverLock.Lock()
	go func() {
		s.srv.Run()
	}()

	t := &mocks.AddOnServer{}
	t.On("Start").Return(nil).Once()
	t.On("Stop").Return(nil).Once()

	s.srv.AddServer(t)

	s.Assert().Equal(1, len(s.srv.otherServers))

	s.srv.Stop()

	s.Assert().NotNil(s.srv)
	s.Assert().FileExists(config.GetPath("params"))

	s.srv.serversLock.Lock()
	s.Assert().Zero(len(s.srv.otherServers))
	s.srv.serversLock.Unlock()

	s.serverLock.Unlock()
}

func (s *ServerTestSuite) Test_AddServer_Started() {
	t := &mocks.AddOnServer{}
	t.On("Start").Return(nil).Once()

	s.srv.AddServer(t)

	s.Assert().Equal(1, len(s.srv.otherServers))
}
