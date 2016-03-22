#!/usr/bin/python3
# -*- coding: utf-8 -*-
from __future__ import with_statement
import paho.mqtt.client as mqtt
import json
import time
import sys

######## r3 ############


def sendR3Message(client, structname, datadict):
    client.publish(structname, json.dumps(datadict))

# Start zmq connection to publish / forward sensor data
client = mqtt.Client()
client.connect("mqtt.realraum.at", 1883, 60)

# listen for sensor data and forward them
if len(sys.argv) < 3:
    sendR3Message(client, "realraum/backdoorcx/gasalert",
                  {"Ts": int(time.time())})
else:
    client.publish(sys.argv[1], sys.argv[2])
client.loop(timeout=1.0, max_packets=1)
client.disconnect()


# {“OnBattery”:bool, PercentBattery:float, LineVoltage: float, LoadPercent: float,
