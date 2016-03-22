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
    sendR3Message(client, "realraum/backdoorcx/powerloss",
                  {"Ts": int(time.time()),
                   "OnBattery": bool(True),
                   "PercentBattery": float(42.0),
                   "LineVoltage": float(2904.0),
                   "LoadPercent": float(0815.0)
                   })
else:
    client.publish(sys.argv[1], sys.argv[2])
client.loop(timeout=1.0, max_packets=1)
client.disconnect()


# {“OnBattery”:bool, PercentBattery:float, LineVoltage: float, LoadPercent: float,
