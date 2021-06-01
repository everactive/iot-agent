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

package config

import (
	"os"
	"path"
	"testing"

	"github.com/snapcore/snapd/osutil"
)

func getExpectedCredentialsPath() string {
	var expectedCredentialsPath string
	if len(os.Getenv(paramsEnvVarOverride)) > 0 {
		expectedCredentialsPath = path.Join(os.Getenv(paramsEnvVarOverride), "/current", DefaultCredentialsPath)
	} else {
		if len(os.Getenv(paramsEnvVar)) > 0 {
			expectedCredentialsPath = path.Join(os.Getenv(paramsEnvVar), "../current", DefaultCredentialsPath)
		}
	}

	return expectedCredentialsPath
}

func TestReadParameters(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"default-settings-create"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			{
				expectedCredentialsPath := getExpectedCredentialsPath()
				got := ReadParameters()
				if got.IdentityURL != DefaultIdentityURL {
					t.Errorf("Config.ReadParameters() got = %v, want %v", got.IdentityURL, DefaultIdentityURL)
				}
				if got.CredentialsPath != expectedCredentialsPath {
					t.Errorf("Config.ReadParameters() got = %v, want %v", got.CredentialsPath, expectedCredentialsPath)
				}

				_ = os.Remove(paramsFilename)
			}
		})
	}
}

func TestRemoveParameters(t *testing.T) {
	tests := []struct {
		name    string
		args    Settings
		wantErr bool
	}{
		{"valid", Settings{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedCredentialsPath := getExpectedCredentialsPath()
			if err := StoreParameters(tt.args); (err != nil) != tt.wantErr {
				t.Errorf("StoreParameters() error = %v, wantErr %v", err, tt.wantErr)
			}

			got := RemoveParameters()
			if got != nil {
				t.Errorf("Config.RemoveParameters() got = %v, want %v", got, nil)
			}

			if osutil.FileExists(GetPath(paramsFilename)) {
				t.Errorf("Config.RemoveParameters() params file still exists after remove")
			}
			if osutil.FileExists(GetPath(expectedCredentialsPath)) {
				t.Errorf("Config.RemoveParameters() expectedCredentialsPath file still exists after remove")
			}
		})
	}
}

func TestStoreParameters(t *testing.T) {
	tests := []struct {
		name    string
		args    Settings
		wantErr bool
	}{
		{"valid", Settings{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedCredentialsPath := getExpectedCredentialsPath()

			if err := StoreParameters(tt.args); (err != nil) != tt.wantErr {
				t.Errorf("StoreParameters() error = %v, wantErr %v", err, tt.wantErr)
			}

			got := ReadParameters()
			if got.IdentityURL != DefaultIdentityURL {
				t.Errorf("Config.ReadParameters() got = %v, want %v", got.IdentityURL, DefaultIdentityURL)
			}
			if got.CredentialsPath != expectedCredentialsPath {
				t.Errorf("Config.ReadParameters() got = %v, want %v", got.CredentialsPath, expectedCredentialsPath)
			}

			_ = os.Remove(paramsFilename)
		})
	}
}
