Sming w2frontdoor Sensor
==============================

Usage
=====



Over-The-Air Update Notes
=========================

Compile using Sming

Update Procedure
----------------

1. configure SPIFFS using ```spiffsconfig.py```
2. ```make clean; make```
3. ```cd out/firmware```
4. start Webserver: ```python -m SimpleHTTPServer 8080```
5. connect via telnet to H801, e.g. ```telnet mashaesp.lan 2323```
6. provide configured auth string: ```auth prevents mistakes <...>```
7. start OTA update: e.g. ```update http://mypc.lan/```
8. terminate telnet session
9. wait
10. power-cycl


