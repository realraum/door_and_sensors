#!/usr/bin/python3
# -*- coding: utf-8 -*-

import json
import time
import paho.mqtt.client as mqtt
import traceback
import Adafruit_BMP.BMP085 as BMP085
import sys


######## r3 MQTT ############

myclientid_ = "printerbone"
sensor = BMP085.BMP085(busnum=2)
query_sensor_intervall_ = 60

def sendR3Message(client, topic, datadict, qos=0, retain=False):
    client.publish(topic, json.dumps(datadict), qos, retain)


def decodeR3Payload(payload):
    try:
        return json.loads(payload.decode("utf-8"))
    except Exception as e:
        print("Error decodeR3Payload:" + str(e))
        return {}


def getAndPublishBMP085SensorValues(client):
    ts=int(time.time())
    sendR3Message(client, "realraum/" + myclientid_ + "/temperature",
                      {"Location": "PrinterBone", "Value": sensor.read_temperature(), "Ts": ts}, retain=True)
    sendR3Message(client, "realraum/" + myclientid_ + "/barometer",
                      {"Location": "PrinterBone", "HPa": sensor.read_pressure()/100.0, "Ts": ts}, retain=True)


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
    client.connect("mqtt.realraum.at", 1883, keepalive=31)
    client.on_disconnect = onMQTTDisconnect
    return client


if __name__ == '__main__':
    client = None
    last_get_sensor_data_ts = 0
    try:
        client = initMQTT()
        while True:
            if time.time() - last_get_sensor_data_ts > query_sensor_intervall_:
                getAndPublishBMP085SensorValues(client)
                last_get_sensor_data_ts = time.time()
            client.loop()

    except Exception as e:
        traceback.print_exc()
    finally:
        if isinstance(client, mqtt.Client):
            # client_stop_loop()
            client.disconnect()
