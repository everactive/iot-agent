package nats

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/everactive/iot-agent/pkg/messages"

	"github.com/everactive/iot-agent/mocks"

	"github.com/stretchr/testify/mock"

	"github.com/everactive/iot-agent/pkg/legacy"

	"github.com/stretchr/testify/assert"
)

func TestServer_SetLegacy(t *testing.T) {

	natsServer := Server{}
	leg := legacy.Handler{}
	var legi legacy.HandlerIFace = &leg

	assert.Nil(t, natsServer.LegacyInterface)
	natsServer.SetLegacy(&legi)
	assert.Equal(t, natsServer.LegacyInterface, &legi)
}

func TestServer_setupSubscriptions_handleMqttConnectionStatus(t *testing.T) {

	conn := mockNatsConnInterface{}
	natsServer := Server{}
	natsServer.encodedConn = &conn

	funcpathruntime := "github.com/everactive/iot-agent/pkg/nats.(*Server).handleMqttConnectionStatus-fm"
	handleMqttConnectionStatus_Subscribed := false

	conn.On("Subscribe", mock.AnythingOfType("string"), mock.Anything).Run(func(args mock.Arguments) {
		funcname := runtime.FuncForPC(reflect.ValueOf(args.Get(1)).Pointer()).Name()
		if funcname == funcpathruntime {
			handleMqttConnectionStatus_Subscribed = true
		}
	}).Return(nil, nil)

	natsServer.setupSubscriptions()

	assert.True(t, handleMqttConnectionStatus_Subscribed)
}

func TestServer_handleMqttConnectionStatus(t *testing.T) {
	conn := mockNatsConnInterface{}
	leg := mocks.HandlerIFace{}
	var legi legacy.HandlerIFace = &leg
	natsServer := Server{}
	natsServer.encodedConn = &conn
	natsServer.SetLegacy(&legi)

	request := &messages.MqttConnectionStatusRequest{}

	responseSuccess := false

	leg.On("IsConnected").Return(true)

	conn.On("Publish", mock.AnythingOfType("string"), mock.AnythingOfType("*messages.MqttConnectionStatus")).Run(func(args mock.Arguments) {
		response := args[1].(*messages.MqttConnectionStatus)
		responseSuccess = response.Connected
	}).Return(nil, nil)

	natsServer.handleMqttConnectionStatus("", "", request)

	assert.True(t, responseSuccess)
	leg.AssertCalled(t, "IsConnected")
}
