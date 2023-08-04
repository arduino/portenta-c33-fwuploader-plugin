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

#include "BlockDevice.h"
#include "MBRBlockDevice.h"
#include "FATFileSystem.h"
#include "certificates.h"
#include "ymodem.h"

BlockDevice* root = BlockDevice::get_default_instance();
MBRBlockDevice sys_bd(root, 1);
FATFileSystem sys_fs("sys");

char filename[256] = {'\0'};

void printError(String msg) {
    Serial.println("ERR:" + msg);
}

void format() {
    MBRBlockDevice::partition(root, 1, 0x0B, 0, 5 * 1024 * 1024);

    int err = sys_fs.reformat(&sys_bd);
    if (err) {
      printError("formatting sys partition");
    }
}

void setup() {
  Serial.begin(115200);
  while (!Serial);

  int err =  sys_fs.mount(&sys_bd);
  if (err) {
    format();
  }

  int chunk_size = 128;
  int byte_count = 0;
  FILE* fp = fopen("/sys/cacert.pem", "wb");

  while (byte_count < cacert_pem_len) {
    if(byte_count + chunk_size > cacert_pem_len)
      chunk_size = cacert_pem_len - byte_count;
    int ret = fwrite(&cacert_pem[byte_count], chunk_size, 1 ,fp);
    if (ret != 1) {
      printError("writing certificates");
      break;
    }
    byte_count += chunk_size;
  }
  fclose(fp);
}

void loop() {
  uint8_t command = 0xFF;

  if (Serial.available()) {
    command = Serial.read();
  }

  if (command == 'Y') {
    FILE* f = fopen("/sys/temp.bin", "wb");
    while (Serial.available()) {
      Serial.read();
    }
    Serial.print("YSTART");
    int ret = Ymodem_Receive(f, 1024 * 1024, filename);
    String name = String(filename);
    if (ret > 0 && name != "") {
      name = "/sys/" + name;
      fclose(f);

      // Delete file having the same name
      struct stat buffer;
      if (stat(name.c_str(), &buffer) == 0) {
          remove(name.c_str());
      }
      ret = rename("/sys/temp.bin", name.c_str());
    }
  }
  if (command == 0xFF) {
    delay(10);
  }
}
