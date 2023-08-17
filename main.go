// portenta-c33-fwuploader-plugin
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
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/arduino/arduino-cli/executils"
	helper "github.com/arduino/fwuploader-plugin-helper"
	"github.com/arduino/go-paths-helper"
	"github.com/arduino/go-xmodem/ymodem"
	"github.com/arduino/uno-r4-wifi-fwuploader-plugin/serial"
	semver "go.bug.st/relaxed-semver"
	serialx "go.bug.st/serial"
	"golang.org/x/exp/slog"
)

const (
	pluginName = "portenta-c33-fwuploader"
)

var (
	versionString = "0.0.0-git"
	commit        = ""
	date          = ""

	//go:embed sketches/reboot/build/arduino.renesas_portenta.portenta_c33/reboot.ino.bin
	rebootSketch []byte
	//go:embed sketches/certificate/build/arduino.renesas_portenta.portenta_c33/certificate.ino.bin
	certificateSketch []byte
	//go:embed sketches/version/build/arduino.renesas_portenta.portenta_c33/version.ino.bin
	versionSketch []byte
)

type portentaC33Plugin struct {
	esptoolBin *paths.Path
	dfuUtilBin *paths.Path
}

func main() {
	esptoolPath, err := helper.FindToolPath("esptool", semver.MustParse("3.3.3"))
	if err != nil {
		fmt.Println("Couldn't find esptool@3.3.3 binary")
		os.Exit(1)
	}
	dfuUtilPath, err := helper.FindToolPath("dfu-util", semver.MustParse("0.11.0-arduino5"))
	if err != nil {
		fmt.Println("Couldn't find dfu-util@0.11.0-arduino5 binary")
		os.Exit(1)
	}
	helper.RunPlugin(&portentaC33Plugin{
		esptoolBin: esptoolPath.Join("esptool"),
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
		return fmt.Errorf("invalid port address")
	}
	if firmwarePath == nil || firmwarePath.IsDir() || !firmwarePath.Exist() {
		return fmt.Errorf("invalid firmware path")
	}
	fmt.Fprintf(feedback.Out(), "Uploading firmware\n")

	rebootFile, err := paths.WriteToTempFile(rebootSketch, paths.TempDir(), "portenta-c33-fwuploader-plugin")
	if err != nil {
		return err
	}
	defer rebootFile.Remove()

	portAddress, err = d.uploadSketch(portAddress, feedback, rebootFile)
	if err != nil {
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
		return fmt.Errorf("invalid port address")
	}
	if certificatePath == nil || certificatePath.IsDir() || !certificatePath.Exist() {
		return fmt.Errorf("invalid certificate path")
	}
	if len(certificatePath.Base()) >= 255 {
		return fmt.Errorf("the certificate name: %v must be less than 256 charaters", certificatePath.Base())
	}
	if certificatePath.Base() == "cacert.pem" {
		return fmt.Errorf("`cacert` name is reserved for the default certificate, please provide a different file name")
	}

	fmt.Fprintf(feedback.Out(), "Uploading certificates to %s...\n", portAddress)

	certificateSketchFile, err := paths.WriteToTempFile(certificateSketch, paths.TempDir(), "portenta-c33-fwuploader-plugin")
	if err != nil {
		return err
	}
	defer certificateSketchFile.Remove()

	portAddress, err = d.uploadSketch(portAddress, feedback, certificateSketchFile)
	if err != nil {
		return err
	}

	// Open connection
	connection, err := serialx.Open(portAddress, &serialx.Mode{BaudRate: 115200})
	if err != nil {
		return err
	}
	if err := connection.SetReadTimeout(20 * time.Second); err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	// Send initial message
	if _, err := connection.Write([]byte("Y\r\n")); err != nil {
		return err
	}

	// reads for possible errors coming from the sketch
	{
		var resultSerialOutput string
		buffer := make([]byte, 64)
		for {
			n, err := connection.Read(buffer)
			if err != nil {
				return err
			}
			if n == 0 {
				break
			}
			resultSerialOutput += string(buffer[0:n])
			if strings.Contains(resultSerialOutput, "YSTART") {
				break
			}
		}

		if strings.Contains(resultSerialOutput, "ERR:") {
			errs := strings.Split(resultSerialOutput, "\r\n")
			result := fmt.Sprintf("%v", errs[1][4:])
			// the ignore the last message as it's not an error
			for _, e := range errs[1 : len(errs)-1] {
				result += ", " + e[4:]
			}
			return errors.New(result)
		}
	}

	data, err := certificatePath.ReadFile()
	if err != nil {
		return err
	}

	fmt.Fprintf(feedback.Out(), "Please wait a few seconds...\n")

	time.Sleep(1 * time.Second)

	if err := ymodem.ModemSend(connection, data, certificatePath.Base()); err != nil {
		return err
	}

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

	versionSketchFile, err := paths.WriteToTempFile(versionSketch, paths.TempDir(), "portenta-c33-fwuploader-plugin")
	if err != nil {
		return nil, err
	}
	defer versionSketchFile.Remove()

	portAddress, err = d.uploadSketch(portAddress, feedback, versionSketchFile)
	if err != nil {
		return nil, err
	}

	port, err := serialx.Open(portAddress, &serialx.Mode{
		BaudRate: 9600,
		Parity:   serialx.NoParity,
		DataBits: 8,
		StopBits: serialx.OneStopBit,
	})
	if err != nil {
		return nil, err
	}
	defer port.Close()

	// wait 1 second to allow the sketch to fill the buffer with the version string
	time.Sleep(time.Second)

	if err := port.SetReadTimeout(time.Second); err != nil {
		return nil, err
	}
	buff := make([]byte, 30)
	n, err := port.Read(buff)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, errors.New("couldn't read serial buffer")
	}

	version := strings.TrimSpace(string(buff[:n]))
	return semver.ParseRelaxed(version), nil
}

func (d *portentaC33Plugin) uploadSketch(portAddress string, feedback *helper.PluginFeedback, sketch *paths.Path) (string, error) {
	slog.Info("upload_sketch")

	// Will be used later to check if the OS changed the serial port.
	allSerialPorts, err := serial.AllPorts()
	if err != nil {
		return "", err
	}

	cmd, err := executils.NewProcess([]string{}, d.dfuUtilBin.String(), "--device", "0x2341:0x0068,:0x0368", "-D", sketch.String(), "-a0", "-Q")
	if err != nil {
		return "", err
	}
	cmd.RedirectStderrTo(feedback.Err())
	cmd.RedirectStdoutTo(feedback.Out())
	if err := cmd.Run(); err != nil {
		return "", err
	}

	// When a board is successfully rebooted in esp32 mode, it might change the serial port.
	// Every 250ms we're watching for new ports, if a new one is found we return that otherwise
	// we'll wait the 10 seconds timeout expiration.
	newPort, changed, err := allSerialPorts.NewPort()
	if err != nil {
		return "", err
	}
	if changed {
		return newPort, nil
	}

	return portAddress, nil
}
