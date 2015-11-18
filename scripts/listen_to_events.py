#!/usr/bin/python3
# -*- coding: utf-8 -*-
import os
import os.path
import sys
import signal
import json
import traceback
import time
import paho.mqtt.client as mqtt
########################

def decodeR3Message(multipart_msg):
    try:
        return (multipart_msg[0], json.loads(multipart_msg[1]))
    except Exception as e:
        logging.debug("decodeR3Message:"+str(e))
        return ("",{})

# The callback for when the client receives a CONNACK response from the server.
def on_connect(client, userdata, flags, rc):
    print("Connected with result code "+str(rc))

    # Subscribing in on_connect() means that if we lose the connection and
    # reconnect then subscriptions will be renewed.
    client.subscribe("#")
    # client.subscribe("$SYS/#")

# The callback for when a PUBLISH message is received from the server.
def on_message(client, userdata, msg):
    print(msg.topic+": %s (%s)" % (msg.payload, type(msg.payload)))
    #(structname, dictdata) = decodeR3Message(data)
    #print("Got data: " + structname + ":"+ str(dictdata))

client = mqtt.Client()
client.on_connect = on_connect
client.on_message = on_message

client.connect("mqtt.realraum.at", 1883, 60)

# Blocking call that processes network traffic, dispatches callbacks and
# handles reconnecting.
# Other loop*() functions are available that give a threaded interface and a
# manual interface.
client.loop_forever()