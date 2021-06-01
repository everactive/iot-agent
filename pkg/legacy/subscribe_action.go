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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/avast/retry-go"
	"github.com/everactive/iot-devicetwin/pkg/messages"
	"github.com/snapcore/snapd/client"

	"github.com/everactive/iot-agent/config"
	"github.com/everactive/iot-agent/snapdapi"
)

// SubscribeAction is the message format for the action topic
type SubscribeAction struct {
	messages.SubscribeAction
}

// Device gets details of the device
func (act *SubscribeAction) Device(orgId, deviceId string) messages.PublishDevice {
	// Call the snapd API for the device information
	info, err := snapd.DeviceInfo()
	if err != nil {
		return messages.PublishDevice{Id: act.Id, Success: false, Message: err.Error()}
	}

	result := messages.Device{
		OrgId:     orgId,
		DeviceId:  deviceId,
		Brand:     info.Brand,
		Model:     info.Model,
		Serial:    info.SerialNumber,
		DeviceKey: info.DeviceKey,
		Store:     info.StoreID,
	}

	// Call the snapd API for the OS version information (ignore errors)
	version, err := act.serverVersion(deviceId)
	if err == nil {
		result.Version = &version
	}

	return messages.PublishDevice{Id: act.Id, Success: true, Result: &result}
}

// SnapInstall installs a new snap
func (act *SubscribeAction) SnapInstall() messages.PublishSnapTask {
	if len(act.Snap) == 0 {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: "No snap name provIded for install"}
	}

	// Call the snapd API
	result, err := snapd.Install(act.Snap, nil)
	if err != nil {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: err.Error()}
	}
	return messages.PublishSnapTask{Id: act.Id, Success: true, Result: result}
}

// SnapRemove removes an existing snap
func (act *SubscribeAction) SnapRemove() messages.PublishSnapTask {
	if len(act.Snap) == 0 {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: "No snap name provIded for remove"}
	}

	// Call the snapd API
	result, err := snapd.Remove(act.Snap, nil)
	if err != nil {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: err.Error()}
	}
	return messages.PublishSnapTask{Id: act.Id, Success: true, Result: result}
}

// SnapList lists installed snaps
func (act *SubscribeAction) SnapList(deviceId string) messages.PublishSnaps {
	// Call the snapd API
	snaps, err := snapd.List([]string{}, nil)
	if err != nil {
		return messages.PublishSnaps{Id: act.Id, Success: false, Message: err.Error()}
	}

	// Convert the snaps into the device twin format
	ss := []*messages.DeviceSnap{}

	for _, s := range snaps {
		// Get the config for the snap (ignore errors)
		var conf string
		c, err := snapd.Conf(s.Name)
		if err == nil {
			resp, err := serializeResponse(c)
			if err == nil {
				conf = string(resp)
			}
		}

		ss = append(ss, &messages.DeviceSnap{
			DeviceId:      deviceId,
			Name:          s.Name,
			InstalledSize: s.InstalledSize,
			InstalledDate: s.InstallDate,
			Status:        s.Status,
			Channel:       s.Channel,
			Confinement:   s.Confinement,
			Version:       s.Version,
			Revision:      s.Revision.N,
			Devmode:       s.DevMode,
			Config:        conf,
		})
	}

	return messages.PublishSnaps{Id: act.Id, Success: true, Result: ss}
}

func (act *SubscribeAction) refreshSnap(name string, opts *client.SnapOptions) messages.PublishSnapTask {
	// Call the snapd API
	result, err := snapd.Refresh(name, opts)
	if err != nil {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: err.Error()}
	}
	return messages.PublishSnapTask{Id: act.Id, Success: true, Result: result}
}

// SnapSwitch refreshes an existing snap
func (act *SubscribeAction) SnapSwitch() messages.PublishSnapTask {
	if len(act.Snap) == 0 {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: "No snap name provIded for switch"}
	}

	if len(act.Data) <= 0 {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: "No channel provided for switch"}
	}

	options := &client.SnapOptions{Channel: act.Data}
	return act.refreshSnap(act.Snap, options)
}

// SnapRefresh refreshes an existing snap
func (act *SubscribeAction) SnapRefresh() messages.PublishSnapTask {
	if len(act.Snap) == 0 {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: "No snap name provIded for refresh"}
	}

	return act.refreshSnap(act.Snap, nil)
}

// SnapRevert reverts an existing snap
func (act *SubscribeAction) SnapRevert() messages.PublishSnapTask {
	if len(act.Snap) == 0 {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: "No snap name provIded for revert"}
	}

	// Call the snapd API
	result, err := snapd.Revert(act.Snap, nil)
	if err != nil {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: err.Error()}
	}
	return messages.PublishSnapTask{Id: act.Id, Success: true, Result: result}
}

// SnapEnable enables an existing snap
func (act *SubscribeAction) SnapEnable() messages.PublishSnapTask {
	if len(act.Snap) == 0 {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: "No snap name provIded for enable"}
	}

	// Call the snapd API
	result, err := snapd.Enable(act.Snap, nil)
	if err != nil {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: err.Error()}
	}
	return messages.PublishSnapTask{Id: act.Id, Success: true, Result: result}
}

// SnapDisable disables an existing snap
func (act *SubscribeAction) SnapDisable() messages.PublishSnapTask {
	if len(act.Snap) == 0 {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: "No snap name provIded for disable"}
	}

	// Call the snapd API
	result, err := snapd.Disable(act.Snap, nil)
	if err != nil {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: err.Error()}
	}
	return messages.PublishSnapTask{Id: act.Id, Success: true, Result: result}
}

// SnapStart sets the config for a snap
func (act *SubscribeAction) SnapStart() messages.PublishSnapTask {
	if len(act.Snap) == 0 {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: "No snap name provided for start"}
	}

	services, err := validateServices(act.Snap, act.Data)
	if err != nil {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: err.Error()}
	}

	// Call the snapd API
	result, err := snapd.Start(services, client.StartOptions{})
	if err != nil {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: err.Error()}
	}

	return messages.PublishSnapTask{Id: act.Id, Success: true, Result: result}
}

// SnapStop sets the config for a snap
func (act *SubscribeAction) SnapStop() messages.PublishSnapTask {
	if len(act.Snap) == 0 {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: "No snap name provided for stop"}
	}

	services, err := validateServices(act.Snap, act.Data)
	if err != nil {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: err.Error()}
	}

	// Call the snapd API
	result, err := snapd.Stop(services, client.StopOptions{})
	if err != nil {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: err.Error()}
	}

	return messages.PublishSnapTask{Id: act.Id, Success: true, Result: result}
}

// SnapRestart sets the config for a snap
func (act *SubscribeAction) SnapRestart() messages.PublishSnapTask {
	if len(act.Snap) == 0 {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: "No snap name provided for restart"}
	}

	services, err := validateServices(act.Snap, act.Data)
	if err != nil {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: err.Error()}
	}

	// Call the snapd API
	result, err := snapd.Restart(services, client.RestartOptions{})
	if err != nil {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: err.Error()}
	}

	return messages.PublishSnapTask{Id: act.Id, Success: true, Result: result}
}

func validateServices(snapName, msg string) ([]string, error) {
	// Deserialize the settings
	var data messages.SnapService
	if err := json.Unmarshal([]byte(msg), &data); err != nil {
		return nil, err
	}

	// If the service list is empty, assume the action is for the whole snap, otherwise assume
	// fully qualified service names like snapname.daemonname
	if len(data.Services) == 0 {
		return []string{snapName}, nil
	}

	for _, s := range data.Services {
		if !strings.HasPrefix(s, snapName) {
			return nil, fmt.Errorf("invalid service: %s", s)
		}
	}
	return data.Services, nil
}

// SnapConf gets the config for a snap
func (act *SubscribeAction) SnapConf() messages.PublishResponse {
	if len(act.Snap) == 0 {
		return messages.PublishResponse{Id: act.Id, Success: false, Message: "No snap name provIded for config"}
	}

	// Call the snapd API
	_, err := snapd.Conf(act.Snap)
	if err != nil {
		return messages.PublishResponse{Id: act.Id, Success: false, Message: err.Error()}
	}

	return messages.PublishResponse{Id: act.Id, Success: true}
}

// SnapSetConf sets the config for a snap
func (act *SubscribeAction) SnapSetConf() messages.PublishSnapTask {
	if len(act.Snap) == 0 {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: "No snap name provIded for set config"}
	}

	// Deserialize the settings
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(act.Data), &data); err != nil {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: err.Error()}
	}

	// Call the snapd API
	result, err := snapd.SetConf(act.Snap, data)
	if err != nil {
		return messages.PublishSnapTask{Id: act.Id, Success: false, Message: err.Error()}
	}

	return messages.PublishSnapTask{Id: act.Id, Success: true, Result: result}
}

// SnapInfo gets the info for a snap
func (act *SubscribeAction) SnapInfo() messages.PublishSnap {
	if len(act.Snap) == 0 {
		return messages.PublishSnap{Id: act.Id, Success: false, Message: "No snap name provIded for snap info"}
	}

	// Call the snapd API
	result, _, err := snapd.Snap(act.Snap)
	if err != nil {
		return messages.PublishSnap{Id: act.Id, Success: false, Message: err.Error()}
	}

	deviceSnap := messages.DeviceSnap{
		Channel:       result.Channel,
		Config:        "",
		Confinement:   result.Confinement,
		DeviceId:      result.ID,
		Devmode:       result.DevMode,
		InstalledDate: result.InstallDate,
		InstalledSize: result.InstalledSize,
		Name:          result.Name,
		Revision:      result.Revision.N,
		Status:        result.Status,
		Version:       result.Version,
	}

	return messages.PublishSnap{Id: act.Id, Success: true, Result: &deviceSnap}
}

// SnapAck adds an assertion to the device
func (act *SubscribeAction) SnapAck() messages.PublishResponse {
	// Call the snapd API
	err := snapd.Ack([]byte(act.Data))
	if err != nil {
		return messages.PublishResponse{Id: act.Id, Success: false, Message: err.Error()}
	}

	return messages.PublishResponse{Id: act.Id, Success: true}
}

// SnapServerVersion gets details of the device
func (act *SubscribeAction) SnapServerVersion(deviceId string) messages.PublishDeviceVersion {
	// Call the snapd API
	result, err := act.serverVersion(deviceId)
	if err != nil {
		return messages.PublishDeviceVersion{Id: act.Id, Success: false, Message: err.Error()}
	}

	return messages.PublishDeviceVersion{Id: act.Id, Success: true, Result: &result}
}

func (act *SubscribeAction) serverVersion(deviceId string) (messages.DeviceVersion, error) {
	// Call the snapd API
	version, err := snapd.ServerVersion()
	if err != nil {
		return messages.DeviceVersion{}, err
	}

	return messages.DeviceVersion{
		DeviceId:      deviceId,
		Version:       version.Version,
		Series:        version.Series,
		OsId:          version.OSID,
		OsVersionId:   version.OSVersionID,
		OnClassic:     version.OnClassic,
		KernelVersion: version.KernelVersion,
	}, nil
}

func (act *SubscribeAction) Unregister(orgId, deviceId string) messages.PublishDevice {
	// Call the snapd API for the device information
	info, err := snapd.DeviceInfo()
	if err != nil {
		return messages.PublishDevice{Id: act.Id, Success: false, Message: err.Error()}
	}

	result := messages.Device{
		OrgId:     orgId,
		DeviceId:  deviceId,
		Brand:     info.Brand,
		Model:     info.Model,
		Serial:    info.SerialNumber,
		DeviceKey: info.DeviceKey,
		Store:     info.StoreID,
	}

	// Call the snapd API for the OS version information (ignore errors)
	version, err := act.serverVersion(deviceId)
	if err == nil {
		result.Version = &version
	}

	// Delete the configuration files to unregister this device
	err = config.RemoveParameters()
	if err != nil {
		return messages.PublishDevice{Id: act.Id, Success: false, Message: err.Error()}
	}

	return messages.PublishDevice{Id: act.Id, Success: true, Result: &result}
}

// RetrieveLogs pulls syslog logs from the snapd api and uploads them to an accessible S3 url
func (act *SubscribeAction) RetrieveLogs(snapd snapdapi.SnapdClient) messages.PublishResponse {

	var data messages.DeviceLogs
	if err := json.Unmarshal([]byte(act.Data), &data); err != nil {
		return messages.PublishResponse{Id: act.Id, Success: false, Message: err.Error()}
	}

	options := client.LogOptions{
		N:      data.Limit,
		Follow: false,
	}

	// call snapd api
	ch, err := snapd.Logs(options)
	if err != nil {
		return messages.PublishResponse{Id: act.Id, Success: false, Message: err.Error()}
	}

	//accumulate and format log strings to upload to S3 url
	var sb strings.Builder
	for msg := range ch {
		sb.WriteString(fmt.Sprintf(
			"[%s] %s %s %s\n",
			msg.Timestamp.String(),
			msg.PID,
			msg.SID,
			msg.Message,
		))
	}
	logs := sb.String()

	req, err := http.NewRequest(http.MethodPut, data.Url, strings.NewReader(logs))
	req.ContentLength = int64(len(logs))
	if err != nil {
		return messages.PublishResponse{Id: act.Id, Success: false, Message: err.Error()}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return messages.PublishResponse{Id: act.Id, Success: false, Message: err.Error()}
	}

	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return messages.PublishResponse{Id: act.Id, Success: false, Message: err.Error()}
		}
		resp_err := fmt.Sprintf("PUT response was non-200 code. code = %d, body = %s", resp.StatusCode, string(body))
		return messages.PublishResponse{Id: act.Id, Success: false, Message: resp_err}
	}

	return messages.PublishResponse{
		Id:      act.Id,
		Action:  act.Action,
		Success: true,
	}
}

// SnapSnapshot creates a snapshot of a snap and uploads it to an S3 url
func (act *SubscribeAction) SnapSnapshot(snapd snapdapi.SnapdClient) messages.PublishResponse {

	var data messages.SnapSnapshot
	if err := json.Unmarshal([]byte(act.Data), &data); err != nil {
		return messages.PublishResponse{Id: act.Id, Success: false, Message: err.Error()}
	}

	snaps := []string{act.Snap}
	setID, _, err := snapd.SnapshotMany(snaps, nil)
	if err != nil {
		return messages.PublishResponse{Id: act.Id, Success: false, Message: err.Error()}
	}

	var body io.ReadCloser
	var length int64

	// Exporting the snapshot may fail because the snapshot isn't ready yet.
	// There isn't a way within the API (that we have found) to detect that the
	// snapshot is ready.
	retryFun := func() error {
		body, length, err = snapd.SnapshotExport(setID)
		return err
	}

	err = retry.Do(retryFun)
	if err != nil {
		return messages.PublishResponse{Id: act.Id, Success: false, Message: err.Error()}
	}

	// upload to url
	req, err := http.NewRequest(http.MethodPut, data.Url, body)
	req.ContentLength = length
	if err != nil {
		return messages.PublishResponse{Id: act.Id, Success: false, Message: err.Error()}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return messages.PublishResponse{Id: act.Id, Success: false, Message: err.Error()}
	}

	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return messages.PublishResponse{Id: act.Id, Success: false, Message: err.Error()}
		}
		resp_err := fmt.Sprintf("PUT response was non-200 code. code = %d, body = %s", resp.StatusCode, string(body))
		return messages.PublishResponse{Id: act.Id, Success: false, Message: resp_err}
	}

	return messages.PublishResponse{
		Id:      act.Id,
		Action:  act.Action,
		Success: true,
	}
}
