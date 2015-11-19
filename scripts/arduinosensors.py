#!/usr/bin/python3
# -*- coding: utf-8 -*-

import json
import time
import serial
import paho.mqtt.client as mqtt
import traceback
######## r3 ZMQ ############

myclientid_="pillar"

def sendR3Message(client, topic, datadict):
    client.publish(topic, json.dumps(datadict))

#Start zmq connection to publish / forward sensor data
def initMQTT():
    client = mqtt.Client(client_id=myclientid_)
    client.connect("mqtt.realraum.at", 1883, 60)
    return client
    
#Initialize TTY interface
def initTTY():
    tty = serial.Serial(port='/dev/ttyUSB0', baudrate=9600,timeout=30)
    tty.flushInput()
    tty.flushOutput()
    return tty
    
#listen for sensor data and forward them    
def handle_sensors(client,tty):
    sensordata = tty.readline()
    if not sensordata is None and len(sensordata) > 2:
        sensordata = sensordata[:-2] 
        if sensordata == b'PanicButton':
            sendR3Message(client,"realraum/"+myclientid_+"/boredoombuttonpressed",{"Ts":int(time.time())})
        elif sensordata == b'movement':
            sendR3Message(client, "realraum/"+myclientid_+"/movement", {"Sensorindex":0, "Ts":int(time.time())})

    tty.write(b'*')
    sensordata = tty.readline()
    sensordata = sensordata[:-2]
    temp = float(sensordata[9:])
    if temp != 0:
        sendR3Message(client, "realraum/"+myclientid_+"/temperature", {"Location":"LoTHR", "Value":temp, "Ts":int(time.time())})

    tty.write(b'?')
    sensordata = tty.readline()
    sensordata = sensordata[:-2]
    light = int(sensordata[9:])
    sendR3Message(client, "realraum/"+myclientid_+"/illumination", {"Location":"LoTHR", "Value":light, "Ts":int(time.time())})

if __name__ == '__main__':
    while True:
        tty = None
        client = None
        try:
            tty = initTTY()
            ## if e.g. ttyUSB0 is not available, then code must not reach this line !!
            ## otherwise we continously try to establish a zmq connection just to close it again
            client = initMQTT()
            while True:
                handle_sensors(client,tty)
        except Exception as e:
            traceback.print_exc()
        finally:
            if tty:
                tty.close()
            if isinstance(client,mqtt.Client):
                client.disconnect()
            time.sleep(5)
