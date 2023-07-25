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
	"log"
	"time"

	"github.com/arduino/arduino-cli/executils"
	helper "github.com/arduino/fwuploader-plugin-helper"
	"github.com/arduino/go-paths-helper"
	"github.com/arduino/portenta-c33-fwuploader-plugin/serial"
	semver "go.bug.st/relaxed-semver"
	"golang.org/x/exp/slog"
)

const (
	pluginName = "portenta-c33-fwuploader"
)

var (
	versionString = "0.0.0-git"
)

type portentaC33Plugin struct {
	esptoolBin *paths.Path
	dfuUtilBin *paths.Path
}

func main() {
	esptoolPath, err := helper.FindToolPath("esptool", semver.MustParse("3.3.2"))
	if err != nil {
		log.Fatalln("Couldn't find esptool@3.3.2 binary")
	}
	dfuUtilPath, err := helper.FindToolPath("dfu-util", semver.MustParse("0.11.0-arduino5"))
	if err != nil {
		log.Fatalln("Couldn't find dfu-util@0.11.0-arduino5 binary")
	}

	helper.RunPlugin(&portentaC33Plugin{
		esptoolBin: esptoolPath.Join(("esptool")),
		dfuUtilBin: dfuUtilPath.Join("dfu-util"),
	})
}

// GetPluginInfo returns information about the plugin
func (d *portentaC33Plugin) GetPluginInfo() *helper.PluginInfo {
	return &helper.PluginInfo{
		Name:    pluginName,
		Version: semver.MustParse(versionString),
	}
}

// UploadFirmware performs a firmware upload on the board
func (d *portentaC33Plugin) UploadFirmware(portAddress, fqbn string, firmwarePath *paths.Path, feedback *helper.PluginFeedback) error {
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
	case "arduino:renesas_portenta:portenta_c33":
		// Do some board specific operations here
		fmt.Fprintf(feedback.Out(), "Uploading firmware for %s \n", fqbn)
	default:
		fmt.Fprintf(feedback.Err(), "FQBN %s not supported by the plugin\n", fqbn)
		return fmt.Errorf("invalid fqbn")
	}

	if err := d.reboot(&portAddress, feedback); err != nil {
		return err
	}

	cmd, err := executils.NewProcess([]string{}, d.esptoolBin.String(), "--chip", "esp32c3", "-p", portAddress, "-b", "230400", "--before=default_reset", "--after=hard_reset", "write_flash", "--flash_mode", "dio", "--flash_freq", "80m", "--flash_size", "4MB", "0x0", firmwarePath.String())
	if err != nil {
		return err
	}
	cmd.RedirectStderrTo(feedback.Err())
	cmd.RedirectStdoutTo(feedback.Out())
	if err := cmd.Run(); err != nil {
		return err
	}

	fmt.Fprintf(feedback.Out(), "Upload completed!\n")
	return nil
}

// UploadCertificate performs a certificate upload on the board
func (d *portentaC33Plugin) UploadCertificate(portAddress, fqbn string, certificatePath *paths.Path, feedback *helper.PluginFeedback) error {
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
func (d *portentaC33Plugin) GetFirmwareVersion(portAddress, fqbn string, feedback *helper.PluginFeedback) (*semver.RelaxedVersion, error) {
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

func (d *portentaC33Plugin) reboot(portAddress *string, feedback *helper.PluginFeedback) error {
	// Will be used later to check if the OS changed the serial port.
	allSerialPorts, err := serial.AllPorts()
	if err != nil {
		return err
	}

	if err := d.uploadCommandsSketch(*portAddress, feedback); err != nil {
		return fmt.Errorf("upload commands sketch: %v", err)
	}

	fmt.Fprintf(feedback.Out(), "\nWaiting to flash the binary...\n")

	slog.Info("check if serial port has changed")
	// When a board is successfully rebooted in esp32 mode, it might change the serial port.
	// Every 250ms we're watching for new ports, if a new one is found we return that otherwise
	// we'll wait the 10 seconds timeout expiration.
	newPort, changed, err := allSerialPorts.NewPort()
	if err != nil {
		return err
	}
	if changed {
		*portAddress = newPort
	}
	return nil
}

func (d *portentaC33Plugin) uploadCommandsSketch(portAddress string, feedback *helper.PluginFeedback) error {
	slog.Info("upload_command_sketch")

	sketch := paths.New("./sketches/C3SerialPassthrough/build/C3SerialPassthrough.ino.bin")

	slog.Info("uploading command sketch with dfu-util")
	cmd, err := executils.NewProcess([]string{}, d.dfuUtilBin.String(), "--device", "0x2341:0x0068,:0x0368", "-D", sketch.String(), "-a0", "-Q")
	if err != nil {
		return err
	}
	cmd.RedirectStderrTo(feedback.Err())
	cmd.RedirectStdoutTo(feedback.Out())
	if err := cmd.Run(); err != nil {
		return err
	}

	time.Sleep(1 * time.Second)
	return nil
}
