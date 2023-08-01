#!/usr/bin/bash

for folder in certificate reboot version
do
    arduino-cli compile -e --profile portenta_c33 $folder
done


