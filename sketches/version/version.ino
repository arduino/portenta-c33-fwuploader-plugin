#include <WiFiC3.h>

void setup() {
  Serial.begin(9600);
  while (!Serial) {
    ; // wait for serial port to connect. Needed for native USB port only
  }

  if (WiFi.status() == WL_NO_MODULE) {
    Serial.println("99.99.99");
    while (true);
  }

  String fv = WiFi.firmwareVersion();
  Serial.println(fv);
}

void loop() {
}
