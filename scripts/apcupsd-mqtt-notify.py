#!/usr/bin/python3
# -*- coding: utf-8 -*-
from __future__ import with_statement
import paho.mqtt.client as mqtt
import json
import time
import sys
import subprocess
import traceback

######## Config File Data Class ############

R3_MQTT_BROKER_HOST = "mqtt.realraum.at"
R3_MQTT_BROKER_PORT = 1883
TOPIC_BACKDOOR_POWERLOSS = "realraum/"+"backdoorcx"+"/powerloss"
TOPIC_BACKDOOR_BATTTEMP = "realraum/"+"backdoorcx"+"/temperature"

######## r3 MQTT ############

def sendR3Message(client, topic, datadict, qos=0, retain=False):
    client.publish(topic, json.dumps(datadict), qos, retain)

######## APCUPSD ############

def getAPCStatus():
    apc_status = {}
    try:
        res = subprocess.check_output("/sbin/apcaccess", timeout=8).decode("utf8")
        for line in res.split("\n"):
            try:
                key,val,*r = line.split(': ')
            except:
                continue
            key = key.rstrip().lower()
            val = val.strip().split(" ")[0]
            apc_status[key] = val
    except Exception as e:
        traceback.print_exc()
    return apc_status


############ Main Routine ############

if __name__ == '__main__':
    client = mqtt.Client()
    client.connect(R3_MQTT_BROKER_HOST, R3_MQTT_BROKER_PORT, 60)
    #listen for sensor data and forward them
    apc_status = getAPCStatus()
    batt_percent = float(apc_status["bcharge"]) if "bcharge" in apc_status else 0
    ts = int(time.time())
    if sys.argv[0].endswith("offbattery"):
        sendR3Message(client,TOPIC_BACKDOOR_POWERLOSS,{ "OnBattery":False,"PercentBattery":batt_percent,"Ts":ts}, qos=2)
    elif sys.argv[0].endswith("onbattery"):
        sendR3Message(client,TOPIC_BACKDOOR_POWERLOSS,{ "OnBattery":True,"PercentBattery":batt_percent,"Ts":ts}, qos=2)
    else:
        sendR3Message(client,TOPIC_BACKDOOR_POWERLOSS,{ "OnBattery":apc_status["status"] != "ONLINE","PercentBattery":batt_percent,"LineVoltage":float(apc_status["linev"]),"LoadPercent":float(apc_status["loadpct"]),"Ts":int(time.time())}, qos=0)
        sendR3Message(client,TOPIC_BACKDOOR_BATTTEMP,{ "Location": "UPS "+apc_status["upsname"]+" Battery", "Value":float(apc_status["itemp"]),"Ts":ts}, qos=0)
    client.loop(timeout=10.0)
    client.disconnect()



