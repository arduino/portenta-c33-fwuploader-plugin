# portenta-c33-fwuploader-plugin

[![Check Go status](https://github.com/arduino/portenta-c33-fwuploader-plugin/actions/workflows/check-go-task.yml/badge.svg)](https://github.com/arduino/portenta-c33-fwuploader-plugin/actions/workflows/check-go-task.yml)

The `portenta-c33-fwuploader-plugin` is a core component of the [arduino-fwuploader](https://github.com/arduino/arduino-fwuploader). The purpose of this plugin is to abstract all the
business logic needed to update firmware and certificates for the [portenta c33](https://docs.arduino.cc/hardware/portenta-c33) board.

## How to contribute

Contributions are welcome!

:sparkles: Thanks to all our [contributors](https://github.com/arduino/portenta-c33-fwuploader-plugin/graphs/contributors)! :sparkles:

### Requirements

1. [Go](https://go.dev/) version 1.20 or later
1. [Task](https://taskfile.dev/) to help you run the most common tasks from the command line
1. The [portenta c33](https://docs.arduino.cc/hardware/portenta-c33) board to test the core parts.

## Development

When running the plugin inside the fwuploader, the required tools are downloaded by the fwuploader. If you run only the plugin, you must provide them by hand.
Therefore be sure to place the `esptool` and `dfu-util` binaries in the correct folders like the following:

```bash
.
├── dfu-util
│   └── 0.11.0-arduino5
│       └── dfu-util
├── esptool
│   └── 3.3.3
│       └── esptool
└── portenta-c33-fwuploader-plugin_linux_amd64
    └── bin
        └── portenta-c33-fwuploader-plugin
```

**Commands**

- `portenta-c33-fwuploader-plugin cert flash -p /dev/ttyACM0 ./certificate/testdata/portenta.pem`
- `portenta-c33-fwuploader-plugin firmware get-version -p /dev/ttyACM0`
- `portenta-c33-fwuploader-plugin firmware flash -p /dev/ttyACM0 ~/Documents/fw0.2.0.bin`

## Security

If you think you found a vulnerability or other security-related bug in the portenta-c33-fwuploader-plugin, please read our [security
policy] and report the bug to our Security Team 🛡️ Thank you!

e-mail contact: security@arduino.cc

## License

portenta-c33-fwuploader-plugin is licensed under the [AGPL 3.0](LICENSE.txt) license.

You can be released from the requirements of the above license by purchasing a commercial license. Buying such a license
is mandatory if you want to modify or otherwise use the software for commercial activities involving the Arduino
software without disclosing the source code of your own applications. To purchase a commercial license, send an email to
license@arduino.cc
