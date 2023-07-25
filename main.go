// fwuploader-plugin-helper
// Copyright (c) 2023 Arduino LLC.  All right reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"time"

	helper "github.com/arduino/fwuploader-plugin-helper"
	"github.com/arduino/go-paths-helper"
	semver "go.bug.st/relaxed-semver"
)

type dummyPlugin struct {
}

func main() {
	helper.RunPlugin(&dummyPlugin{})
}

// GetPluginInfo returns information about the plugin
func (d *dummyPlugin) GetPluginInfo() *helper.PluginInfo {
	return &helper.PluginInfo{
		Name:    "dummy",
		Version: semver.MustParse("1.0.0"),
	}
}

// UploadFirmware performs a firmware upload on the board
func (d *dummyPlugin) UploadFirmware(portAddress, fqbn string, firmwarePath *paths.Path, feedback *helper.PluginFeedback) error {
	if portAddress == "" {
		fmt.Fprintln(feedback.Err(), "Port address not specified")
		return fmt.Errorf("invalid port address")
	}
	fmt.Fprintf(feedback.Out(), "Uploading %s to %s...\n", firmwarePath, portAddress)
	if fqbn == "" {
		fmt.Fprintln(feedback.Err(), "FQBN not specified")
		return fmt.Errorf("invalid fqbn")
	}

	// Providing the fqbn to the plugin allows us to support a family of boards instead of a single one
	switch fqbn {
	case "arduino:renesas_uno:unor5":
		// Do some board specific operations here
		fmt.Fprintf(feedback.Out(), "Uploading firmware for %s \n", fqbn)
	case "arduino:renesas_uno:unor4wifi":
		// Do some board specific operations here
		fmt.Fprintf(feedback.Out(), "Uploading firmware for %s \n", fqbn)
	default:
		fmt.Fprintf(feedback.Err(), "FQBN %s not supported by the plugin\n", fqbn)
		return fmt.Errorf("invalid fqbn")
	}

	// Fake upload
	time.Sleep(5 * time.Second)

	fmt.Fprintf(feedback.Out(), "Upload completed!\n")
	return nil
}

// UploadCertificate performs a certificate upload on the board
func (d *dummyPlugin) UploadCertificate(portAddress, fqbn string, certificatePath *paths.Path, feedback *helper.PluginFeedback) error {
	if portAddress == "" {
		fmt.Fprintln(feedback.Err(), "Port address not specified")
		return fmt.Errorf("invalid port address")
	}
	fmt.Fprintf(feedback.Out(), "Uploading certificates to %s...\n", portAddress)

	// Fake upload
	time.Sleep(5 * time.Second)

	fmt.Fprintf(feedback.Out(), "Upload completed!\n")
	return nil
}

// GetFirmwareVersion retrieve the firmware version installed on the board
func (d *dummyPlugin) GetFirmwareVersion(portAddress, fqbn string, feedback *helper.PluginFeedback) (*semver.RelaxedVersion, error) {
	if portAddress == "" {
		fmt.Fprintln(feedback.Err(), "Port address not specified")
		return nil, fmt.Errorf("invalid port address")
	}
	fmt.Fprintf(feedback.Out(), "Getting firmware version from %s...\n", portAddress)

	// Fake query
	time.Sleep(5 * time.Second)

	fmt.Fprintf(feedback.Out(), "Done!\n")
	return semver.ParseRelaxed("1.0.0"), nil
}
