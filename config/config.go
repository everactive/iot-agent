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
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/snapcore/snapd/osutil"
)

// Default parameters
const (
	DefaultIdentityURL     = "http://localhost:8030"
	DefaultCredentialsPath = ".secret"
	paramsEnvVar           = "SNAP_DATA"
	paramsEnvVarOverride   = "OVERRIDE_SNAP_DATA"
	paramsFilename         = "params"
)

// Settings defines the application configuration
type Settings struct {
	IdentityURL     string `json:"url"`
	CredentialsPath string `json:"path"`
}

// ReadParameters fetches the store config parameters
func ReadParameters() *Settings {
	Config := Settings{
		IdentityURL:     DefaultIdentityURL,
		CredentialsPath: GetPath(DefaultCredentialsPath),
	}

	p := GetPath(paramsFilename)

	dat, err := ioutil.ReadFile(p)
	if err != nil {
		log.Printf("Error reading config parameters: %v", err)
		_ = StoreParameters(Config)
		return &Config
	}

	err = json.Unmarshal(dat, &Config)
	if err != nil {
		log.Printf("Error parsing config parameters: %v", err)
		return &Config
	}

	return &Config
}

// StoreParameters stores the configuration parameters on the filesystem
func StoreParameters(c Settings) error {
	p := GetPath(paramsFilename)

	// Default empty parameters
	if len(c.IdentityURL) == 0 {
		c.IdentityURL = DefaultIdentityURL
	}
	if len(c.CredentialsPath) == 0 {
		c.CredentialsPath = GetPath(DefaultCredentialsPath)
	}

	// Create the output file
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()

	// Convert the parameters to JSON
	b, err := json.Marshal(c)
	if err != nil {
		log.Printf("Error marshalling config parameters: %v", err)
		return err
	}

	// Output the JSON to the file
	_, err = f.Write(b)
	if err != nil {
		log.Printf("Error storing config parameters: %v", err)
		return err
	}
	_ = f.Sync()

	// Restrict access to the file
	return os.Chmod(p, 0600)
}

// RemoveParameters removes the configuration parameters on the filesystem, effectively unregistering this device
func RemoveParameters() error {
	var err error = nil
	if osutil.FileExists(GetPath(paramsFilename)) {
		err = os.Remove(GetPath(paramsFilename))
		if err != nil {
			return err
		}
	}

	if osutil.FileExists(GetPath(DefaultCredentialsPath)) {
		err = os.Remove(GetPath(DefaultCredentialsPath))
	}

	return err
}

// GetPath returns the full path to the data file
func GetPath(filename string) string {
	if len(os.Getenv(paramsEnvVarOverride)) > 0 {
		return path.Join(os.Getenv(paramsEnvVarOverride), "/current", filename)
	} else {
		if len(os.Getenv(paramsEnvVar)) > 0 {
			return path.Join(os.Getenv(paramsEnvVar), "../current", filename)
		}
	}
	return filename
}
