#!/usr/bin/python3
# -*- coding: utf-8 -*-

import json
import time
import serial
import paho.mqtt.client as mqtt
import traceback
import sys
######## r3 ZMQ ############

myclientid_="xbee"
mylocation_="Outside"
rf433_send_delay_s_ = 0.0

def sendR3Message(client, topic, datadict, qos=0, retain=False):
    client.publish(topic, json.dumps(datadict), qos, retain)

def decodeR3Payload(payload):
    try:
        return json.loads(payload.decode("utf-8"))
    except Exception as e:
        print("Error decodeR3Payload:"+str(e))
        return {}

#Start zmq connection to publish / forward sensor data
def initMQTT():
    client = mqtt.Client(client_id=myclientid_)
    client.connect("mqtt.realraum.at", 1883, keepalive=50)
    return client

#Initialize TTY interface
def initTTY(port):
    tty = serial.Serial(port=port, baudrate=9600,timeout=5 )
    tty.flushInput()
    tty.flushOutput()
    return tty

def handle_arduino_output(client,tty):
    str_humidity = b'Humidity (%): '
    str_temperature = b'Temperature (C): '
    sensordata = tty.readline()
    if sensordata is None or len(sensordata) <= 5:
        return
    if sensordata.startswith(str_humidity):
        humidity = float(sensordata[len(str_humidity):])
        sendR3Message(client,"realraum/"+myclientid_+"/relhumidity",{"Location":mylocation_,"Percent":humidity,"Ts":int(time.time())}, retain=True)
    elif sensordata.startswith(str_temperature):
        temp = float(sensordata[len(str_temperature):])
        sendR3Message(client,"realraum/"+myclientid_+"/temperature",{"Location":mylocation_,"Value":temp,"Ts":int(time.time())}, retain=True)

if __name__ == '__main__':
    tty = None
    client = None
    try:
        tty = initTTY('/dev/ttyUSB1' if len(sys.argv) < 2 else sys.argv[-1])
        ## if e.g. ttyUSB0 is not available, then code must not reach this line !!
        ## otherwise we continously try to establish a zmq connection just to close it again
        client = initMQTT()
        #client.start_loop()
        while True:
            handle_arduino_output(client,tty)
            client.loop()

    except Exception as e:
        traceback.print_exc()
    finally:
        if tty:
            tty.close()
        if isinstance(client,mqtt.Client):
            #client_stop_loop()
            client.disconnect()
