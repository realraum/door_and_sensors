#!/usr/bin/python3
# -*- coding: utf-8 -*-

import json
import time
import serial
import paho.mqtt.client as mqtt
import traceback
import sys
######## r3 ZMQ ############

myclientid_ = "xbee"
mylocation0_ = "Outside"
mylocation1_ = "WC"
rf433_send_delay_s_ = 0.0

def sendR3Message(client, topic, datadict, qos=0, retain=False):
    client.publish(topic, json.dumps(datadict), qos, retain)

def decodeR3Payload(payload):
    try:
        return json.loads(payload.decode("utf-8"))
    except Exception as e:
        print("Error decodeR3Payload:" + str(e))
        return {}

def onMQTTDisconnect(mqttc, userdata, rc):
    if rc != 0:
        print("Unexpected disconnection.")
        while True:
            time.sleep(5)
            print("Attempting reconnect")
            try:
                mqttc.reconnect()
                break
            except ConnectionRefusedError:
                continue
    else:
        print("Clean disconnect.")
        sys.exit()

# Start zmq connection to publish / forward sensor data
def initMQTT():
    client = mqtt.Client(client_id=myclientid_)
    client.connect("mqtt.realraum.at", 1883, keepalive=50)
    client.on_disconnect = onMQTTDisconnect
    return client

# Initialize TTY interface
def initTTY(port):
    tty = serial.Serial(port=port, baudrate=9600,timeout=5 )
    tty.flushInput()
    tty.flushOutput()
    return tty

def publishHumidity(client, datastr, location):
    humidity = float(datastr)
    sendR3Message(client,
        "realraum/" + myclientid_ + "/relhumidity",
            {"Location": location, "Percent": humidity, "Ts": int(time.time())},
         retain=True)

def publishTemperature(client, datastr, location):
    temp = float(datastr)
    sendR3Message(client,
                  "realraum/" + myclientid_ + "/temperature",
                  {"Location": location,
                   "Value": temp,
                   "Ts": int(time.time())},
                  retain=True)

def publishVoltage(client, datastr, location):
    volt = float(datastr)
    minv=3.0
    maxv=3.90
    sendR3Message(client,
                  "realraum/" + myclientid_ + "/voltage",
                  {"Location": location,
                   "Value": volt,
                   "Min": minv,
                   "Max": maxv,
                   "Percent": min(round(100.0 * ((volt - minv) / (maxv-minv)),2),100.0),
                   "Ts": int(time.time())},
                  retain=True)

def handle_arduino_output(client, tty):
    str_humidity0 = b'Humidity 0 (%): '
    str_temperature0 = b'Temperature 0 (C): '
    str_voltage0 = b'Battery Voltage 0 (V): '
    str_humidity1 = b'Humidity 1 (%): '
    str_temperature1 = b'Temperature 1 (C): '
    str_voltage1 = b'Battery Voltage 1 (V): '
    sensordata = tty.readline()
    if sensordata is None or len(sensordata) <= 5:
        return
    if sensordata.startswith(str_humidity0):
        publishHumidity(client, sensordata[len(str_humidity0):], mylocation0_)
    elif sensordata.startswith(str_temperature0):
        publishTemperature(client, sensordata[len(str_temperature0):], mylocation0_)
    elif sensordata.startswith(str_voltage0):
        publishVoltage(client, sensordata[len(str_voltage0):], mylocation0_)
    elif sensordata.startswith(str_humidity1):
        publishHumidity(client, sensordata[len(str_humidity1):], mylocation1_)
    elif sensordata.startswith(str_temperature1):
        publishTemperature(client, sensordata[len(str_temperature1):], mylocation1_)
    elif sensordata.startswith(str_voltage1):
        publishVoltage(client, sensordata[len(str_voltage1):], mylocation1_)

if __name__ == '__main__':
    tty = None
    client = None
    try:
        tty = initTTY('/dev/ttyXBEE' if len(sys.argv) < 2 else sys.argv[-1])
        # if e.g. ttyUSB0 is not available, then code must not reach this line !!
        # otherwise we continously try to establish a zmq connection just to
        # close it again
        client = initMQTT()
        # client.start_loop()
        while True:
            handle_arduino_output(client, tty)
            client.loop()

    except Exception as e:
        traceback.print_exc()
    finally:
        if tty:
            tty.close()
        if isinstance(client, mqtt.Client):
            # client_stop_loop()
            client.disconnect()
