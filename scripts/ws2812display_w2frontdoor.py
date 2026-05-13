#!/usr/bin/python3
# pip install paho-mqtt
# pip install pyserial

__author__ = "ruru"

import serial
import paho.mqtt.client as mqtt
import json

ser = serial.Serial('/dev/ttyLEDs')

def decodeR3Message(topic, data):
    try:
        return (topic, json.loads(data.decode("utf-8")))
    except Exception as e:
        return ("", {})

def on_connect(client, userdata, flags, reason_code, properties):
    if reason_code.is_failure:
        print(f"Failed to connect: {reason_code}")
    else:
        client.subscribe("realraum/w2frontdoor/lock")

def on_message(client, userdata, msg):
    topic, jsondata = decodeR3Message(msg.topic, msg.payload)
    print(topic, jsondata)

    if jsondata.get("Locked"):
        ser.write(b'c')
    else:
        ser.write(b'o')

client = mqtt.Client(mqtt.CallbackAPIVersion.VERSION2)
client.on_connect = on_connect
client.on_message = on_message

client.connect("mqtt.realraum.at", 1883, 60)

client.loop_forever()
