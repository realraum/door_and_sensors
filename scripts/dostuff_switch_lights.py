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
time_schedule_sonoff_ = []

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

def switchname(client,name,action):
    if not isinstance(action,str):
        action = "on" if action else "off"
    if '"' in action:
        return
    if isinstance(name,list):
        for n in name:
            client.publish("action/GoLightCtrl/"+n,'{"Action":"'+action+'"}');
    else:
        client.publish("action/GoLightCtrl/"+name,'{"Action":"'+action+'"}');

def switchsonoff(client,name,action):
    if not isinstance(action,str):
        action = "on" if action else "off"
    if '"' in action:
        return
    if isinstance(name,list):
        for n in name:
            client.publish("action/%s/power" % n, action)
    else:
        client.publish("action/%s/power" % name, action)


def scheduleSwitchSonoff(name,action,time):
    global time_schedule_sonoff_
    ##remove all earlier action entries for names, regardless of action off or on
    for stime, (sname, saction) in time_schedule_sonoff_:
        if stime <= time:
            sname[:] = [ x for x in sname if not x in name]
    ## add new scheduled action for names
    time_schedule_sonoff_.append((time,(name,action)))

def runScheduledEvents(client):
    global time_schedule_sonoff_
    curtime = time.time()
    time_schedule_sonoff_.sort()
    idx=0
    for stime, (sname, saction) in time_schedule_sonoff_:
        if stime > curtime:
            break
        switchsonoff(client,sname,saction)
        idx+=1
    time_schedule_sonoff_ = time_schedule_sonoff_[idx:]


def onLoop(client):
    global last_masha_movement_
    ## run schedules events
    runScheduledEvents(client)
    ## if more than 6 minutes no movement in masha ... switch off light
    if last_masha_movement_ > 0 and time.time() - last_masha_movement_ > 360.0:
        last_masha_movement_ = 0
        #print(last_masha_movement_)
        switchname(client,["mashadecke"],"off")


def signal_handler(self, signal, frame):
    global keep_running_
    print('You pressed Ctrl+C!',file=sys.stderr)
    keep_running_=False

def onMqttMessage(client, userdata, msg):
    global last_status, unixts_panic_button, unixts_last_movement, unixts_last_presence, last_havesunlight_state_, sunlight_change_direction_counter_, last_masha_movement_
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
            if not last_status["Present"]:
                return  # no use switching lights if nobody is here
            # if people are present and the sun is down, switch on CX Lights
            if didSunChangeRecently():
                if isTheSunDown():
                    switchname(client,["cxleds","bluebar","couchwhite","logo","laserball"],"on")
                    switchsonoff(client,["couchred"],"on")
                else:
                    #leave cxleads on, otherwise people will use the ceiling light in CX
                    switchname(client,["bluebar","couchwhite","laserball","logo"],"off")
                    switchsonoff(client,["couchred"],"off")
        elif topic.endswith("/presence") and "Present" in dictdata and "InSpace1" in dictdata:
            if msg.retain:
                last_status = dictdata.copy()
                return  # do not act on retained messages
            # if something changed
            if last_status["Present"] != dictdata["Present"]:
                last_status = dictdata.copy()
                if dictdata["Present"] == True:
                    # someone just arrived
                    # power to tesla labortisch so people can switch on the individual lights (and switch off after everybody leaves)
                    # boiler needs power, so always off. to be switched on manuall when needed
                    switchname(client,["cxleds","boilerolga"],"on")
                    switchsonoff(client,["tesla","lothrboiler","olgaboiler"],"on")
                    if isTheSunDown():
                        switchname(client,["floodtesla","bluebar","couchwhite","laserball","logo"],"on")
                        switchsonoff(client,["couchred"],"on")
                        client.publish("action/ceilingscripts/activatescript",'{"script":"redshift","participating":["ceiling1","ceiling3"],"value":0.7}')
                        # client.publish("action/ceiling1/light",'{"r":400,"b":0,"ww":800,"cw":0,"g":0,"fade":{}}')
                        # client.publish("action/ceiling3/light",'{"r":400,"b":0,"ww":800,"cw":0,"g":0,"fade":{}}')
                    # doppelt hält besser, für die essentiellen dinge
                    switchname(client,["boilerolga","cxleds"],"on")
                else:
                    # everybody left
                    client.publish("action/ceilingscripts/activatescript",'{"script":"off"}')
                    client.publish("action/ceilingAll/light",'{"r":0,"b":0,"ww":0,"cw":0,"g":0,"uv":0,"fade":{}}')
                    switchname(client,["abwasch","couchwhite","laserball","logo","all"],"off")
                    switchsonoff(client,["couchred","tesla","lothrboiler","olgaboiler"],"off")
                    time.sleep(4)
                    switchname(client,["all"],"off")
                    # doppelt hält besser, für die essentiellen dinge
                    client.publish("action/ceilingAll/light",'{"r":0,"b":0,"ww":0,"cw":0,"g":0,"uv":0}')
                    switchname(client,["boilerolga"],"off")
            elif last_status["InSpace1"] != dictdata["InSpace1"] and dictdata["Present"] == True:
                if dictdata["InSpace1"] == True:
                    switchsonoff(client,["couchred"],"on")
                else:
                    switchname(client,["couchwhite","mashadecke","floodtesla"],"off")
                    switchsonoff(client,["couchred"],"off")
            elif last_status["InSpace2"] != dictdata["InSpace2"] and dictdata["Present"] == True:
                if dictdata["InSpace2"] == True:
                    pass
                else:
                    pass
        elif topic.endswith("realraum/xbee/masha/movement"):
            last_masha_movement_=time.time()
            #print(last_masha_movement_)
        elif topic.endswith("/boredoombuttonpressed"):
            pass
        elif topic.endswith("realraum/w2frontdoor/lock"):
            if msg.retain:
                return
            if isTheSunDown() and dictdata["Locked"] == True:
                ## configure hallwaylight to safety switch off after 100s
                client.publish("action/hallwaylight/PulseTime", "%d" % (100+100))
                ## switch on hallwaylight
                switchsonoff(client,["hallwaylight"],"on")
                ## for 30s
                scheduleSwitchSonoff(["hallwaylight"],"off",time.time()+30)
        elif topic.endswith("/ajar"):
            if msg.retain:
                return
            if isTheSunDown() and dictdata["Shut"] == False:
                ## configure hallwaylight to safety switch off after 100s
                client.publish("action/hallwaylight/PulseTime", "%d" % (100+100))
                ## switch on hallwaylight
                switchsonoff(client,["hallwaylight"],"on")
                ## for 30s
                scheduleSwitchSonoff(["hallwaylight"],"off",time.time()+30)
                if topic.endswith("/backdoorcx/ajar"):
                    ## also switch CX light on and leave them on
                    switchname(client,["cxleds"],"on")

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
    last_status = {}
    unixts_panic_button = None
    unixts_last_movement = 0
    unixts_last_presence = 0
    client = mqtt.Client(client_id=os.path.basename(sys.argv[0]))
    client.on_connect = lambda client, userdata, flags, rc: client.subscribe([
        ("realraum/metaevt/presence", 1),
        ("realraum/metaevt/duskordawn", 1),
        ("realraum/pillar/boredoombuttonpressed", 1),
        ("realraum/xbee/masha/movement",1),
        ("realraum/+/ajar",1),
        ("realraum/w2frontdoor/lock",1),
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
