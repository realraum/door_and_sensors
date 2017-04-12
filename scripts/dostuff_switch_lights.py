#!/usr/bin/python3
# -*- coding: utf-8 -*-
import sys
import os
#import threading
import time
import traceback
import paho.mqtt.client as mqtt
import json
import urllib.request
import urllib.parse
import urllib.error

last_havesunlight_state_ = False
sunlight_change_direction_counter_ = 0
last_masha_movement_ = 0
keep_running_ = True

def isTheSunDown():
    return not last_havesunlight_state_

# note that during dusk / dawn several events are fired, so we have this function to 
# easily limit our light change attempts to 2 at 0 and 1
def didSunChangeRecently():
    return sunlight_change_direction_counter_ <= 1

def decodeR3Message(topic, data):
    try:
        return (topic, json.loads(data.decode("utf-8")))
    except Exception as e:
        # print("decodeR3Message:"+str(e))
        return ("", {})


def touchURL(url):
    try:
        urllib.request.urlcleanup()
        f = urllib.request.urlopen(url, timeout=2)
        rq_response = f.read()
        #print("touchURL: url: "+url)
        f.close()
        return rq_response
    except Exception as e:
        print("touchURL: " + str(e))
        return None

def onLoop(client):
    global last_masha_movement_
    ## if more than 10 minutes no movement in masha ... switch off light
    if last_masha_movement_ > 0 and time.time() - last_masha_movement_ > 600.0:
        last_masha_movement_ = 0
        client.publish("action/GoLightCtrl/name",'{"Name":"mashadecke","Action":"off"}')
        touchURL("http://licht.realraum.at/cgi-bin/mswitch.cgi?mashadecke=0")


def signal_handler(self, signal, frame):
    global keep_running_
    print('You pressed Ctrl+C!',file=sys.stderr)
    keep_running_=False

def onMqttMessage(client, userdata, msg):
    global last_status, last_user, unixts_panic_button, unixts_last_movement, unixts_last_presence, last_havesunlight_state_, sunlight_change_direction_counter_
    try:
        (topic, dictdata) = decodeR3Message(msg.topic, msg.payload)
        #print("Got data: " + topic + ":"+ str(dictdata))
        if topic.endswith("/duskordawn") and "HaveSunlight" in dictdata:
            if msg.retain or last_havesunlight_state_ != bool(dictdata["HaveSunlight"]):
                sunlight_change_direction_counter_ = 0
            else:
                sunlight_change_direction_counter_ += 1
            last_havesunlight_state_ = bool(dictdata["HaveSunlight"])
            if msg.retain:
                return  # do not act on retained messages
            if not last_status:
                return  # no use switching lights if nobody is here
            # if people are present and the sun is down, switch on CX Lights
            if didSunChangeRecently():
                if isTheSunDown():
                    touchURL("http://licht.realraum.at/cgi-bin/mswitch.cgi?cxleds=1&bluebar=1&couchred=1&couchwhite=1")
                else:
                    touchURL("http://licht.realraum.at/cgi-bin/mswitch.cgi?cxleds=0&bluebar=0&couchred=0&couchwhite=0")
        elif topic.endswith("/presence") and "Present" in dictdata:
            if msg.retain:
                last_status = dictdata["Present"]
                return  # do not act on retained messages
            # if something changed
            if last_status != dictdata["Present"]:
                last_status = dictdata["Present"]
                if dictdata["Present"] == True:
                    # someone just arrived
                    # power to labortisch so people can switch on the individual lights (and switch off after everybody leaves)
                    # boiler needs power, so always off. to be switched on manuall when needed
                    touchURL(
                        "http://licht.realraum.at/cgi-bin/mswitch.cgi?labortisch=1&cxleds=1&boiler=0&boilerolga=1")
                    if isTheSunDown():
                        touchURL(
                            "http://licht.realraum.at/cgi-bin/mswitch.cgi?basiclight4=1&basiclight1=1&couchred=1&bluebar=1&couchwhite=1&abwasch=1&floodtesla=1")
                    # doppelt hält besser, für die essentiellen dinge
                    touchURL(
                        "http://licht.realraum.at/cgi-bin/mswitch.cgi?boiler=0&labortisch=1&boilerolga=1")
                else:
                    # everybody left
                    client.publish("action/ceilingscripts/activatescript",'{"script":"off"}')
                    client.publish("action/ceilingAll/light",'{"r":0,"b":0,"ww":0,"cw":0,"g":0,"fade":{}}')
                    touchURL(
                        "http://licht.realraum.at/cgi-bin/mswitch.cgi?couchred=0&all=0")
                    time.sleep(2)
                    touchURL("http://licht.realraum.at/cgi-bin/mswitch.cgi?all=0")
                    time.sleep(2)
                    # doppelt hält besser, für die essentiellen dinge
                    client.publish("action/ceilingAll/light",'{"r":0,"b":0,"ww":0,"cw":0,"g":0}')
                    touchURL(
                        "http://licht.realraum.at/cgi-bin/mswitch.cgi?labortisch=0&boiler=0&boilerolga=0")
        elif topic.endswith("realraum/mashaesp/movement"):
            last_masha_movement_=time.time()
        elif topic.endswith("/boredoombuttonpressed"):
            pass

    except Exception as ex:
        print("onMqttMessage: " + str(ex))
        traceback.print_exc(file=sys.stdout)
        sys.exit(1)

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

if __name__ == "__main__":
    last_status = None
    last_user = None
    unixts_panic_button = None
    unixts_last_movement = 0
    unixts_last_presence = 0
    client = mqtt.Client(client_id=os.path.basename(sys.argv[0]))
    client.on_connect = lambda client, userdata, flags, rc: client.subscribe([
        ("realraum/metaevt/presence", 1),
        ("realraum/metaevt/duskordawn", 1),
        ("realraum/pillar/boredoombuttonpressed", 1),
    ])
    client.on_message = onMqttMessage
    client.connect("mqtt.realraum.at", 1883, keepalive=45)
    client.on_disconnect = onMQTTDisconnect

    # Blocking call that processes network traffic, dispatches callbacks and
    # handles reconnecting.
    # Other loop*() functions are available that give a threaded interface and a
    # manual interface.
    while keep_running_:
        client.loop()
        onLoop(client)
        time.sleep(0.2)
