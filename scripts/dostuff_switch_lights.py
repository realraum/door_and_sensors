#!/usr/bin/python3
# -*- coding: utf-8 -*-

##
## TODO: replace this script with Homeautomation webfrontend
##       or rewrite it as "Match Room State -> Ensure Light State"
## 

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

topic_tradfri_onoff_lothr="zigbee2mqtt/w1/TradfriOnOffc9ed"

last_havesunlight_state_ = False
sunlight_change_direction_counter_ = 0
last_masha_no_more_movement_ = 1
last_masha_no_more_movement2_ = 1
last_masha_turned_light_off_by_script_ = 0
masha_ceiling_light_timeout_seconds_ = 660.0
retro_corner_outletplug_timeout_seconds_ = 60*100 # 1h40m
last_w2_locked_ = 0
keep_running_ = True
time_schedule_sonoff_ = [] # | list[tuple[float,tuple[str,str]]]
w1_frontdoor_locked = None
backdoorblue_locked = None

def isTheSunDown(): #->bool :
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
            client.publish("action/GoLightCtrl/"+n,'{"Action":"'+action+'"}', qos=2);
    else:
        client.publish("action/GoLightCtrl/"+name,'{"Action":"'+action+'"}', qos=2);

def switchsonoff(client,name,action):
    if not isinstance(action,str):
        action = "on" if action else "off"
    if '"' in action:
        return
    if isinstance(name,list):
        for n in name:
            client.publish("action/%s/power" % n, action, qos=1)
    else:
        client.publish("action/%s/power" % name, action, qos=1)

def switchesphome(client,name,action):
    if not isinstance(action,str):
        action = "ON" if action else "OFF"
    if '"' in action:
        return
    action = "{\"state\":\""+action.upper()+"\"}"
    if not isinstance(name,list):
        name = [name]
    for n in name:
        client.publish("action/%s/command" % n, action, qos=1)

def switchZigbeeOutlet(client,whg_and_friendlyname,action):
    if not isinstance(action, str) or not action in ["ON","OFF"]:
        action = "ON" if action else "OFF"
    if '"' in action:
        return
    if not isinstance(whg_and_friendlyname,list):
        whg_and_friendlyname = [whg_and_friendlyname]
    for n in whg_and_friendlyname:
        if not (n.startswith("w1/") or n.startswith("w2/")):
            continue
        client.publish("zigbee2mqtt/%s/set/state" % n, action, qos=1)

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

next_run_pulsetime=0.0
def runRegularEvents(client):
    global next_run_pulsetime
    if time.time() > next_run_pulsetime:
        ## configure hallwaylight to safety switch off after 100s
        client.publish("action/hallwaylight/PulseTime1", "%d" % (100+100), qos=2)
        next_run_pulsetime=time.time()+3600*12

def onLoop(client):
    global last_masha_turned_light_off_by_script_, last_w2_locked_
    ## run schedules events
    runScheduledEvents(client)
    runRegularEvents(client)
    ## if more than 11 minutes no movement in masha ... switch off light
    if last_masha_no_more_movement_ > 0 and time.time() - last_masha_no_more_movement_ > masha_ceiling_light_timeout_seconds_ and last_masha_no_more_movement2_ > 0 and time.time() - last_masha_no_more_movement2_ > masha_ceiling_light_timeout_seconds_ and last_masha_turned_light_off_by_script_ < max(last_masha_no_more_movement2_,last_masha_no_more_movement_):
        last_masha_turned_light_off_by_script_ = time.time()
        #print(last_masha_no_more_movement_)
        switchsonoff(client,["mashadecke"],"off")
    if last_w2_locked_ > 0 and time.time() - last_w2_locked_ > retro_corner_outletplug_timeout_seconds_:
        last_w2_locked_ = 0
        switchsonoff(client,["retrocorner"],"off")


def signal_handler(self, signal, frame):
    global keep_running_
    print('You pressed Ctrl+C!',file=sys.stderr)
    keep_running_=False

def onMqttMessage(client, userdata, msg):
    global last_status, unixts_panic_button, unixts_last_movement, unixts_last_presence, last_havesunlight_state_, sunlight_change_direction_counter_, last_masha_no_more_movement_, last_w2_locked_, w1_frontdoor_locked, backdoorblue_locked
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
                    switchname(client,["cxleds","couchwhite","logo","laserball"],"on")
                    switchZigbeeOutlet(client,["w1/OutletBlueLEDBar","w1/OutletAuslageW1"],"ON")
                    switchsonoff(client,["couchred"],"on")
                else:
                    #leave cxleads on, otherwise people will use the ceiling light in CX
                    switchname(client,["couchwhite","laserball","logo"],"off")
                    switchZigbeeOutlet(client,["w1/OutletBlueLEDBar","w1/OutletAuslageW1"],"OFF")
                    switchsonoff(client,["couchred"],"off")
                    switchesphome(client,["subtable"],"off")
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
                    switchsonoff(client,["lothrboiler","olgaboiler"],"on")
                    switchesphome(client,["mashacompressor"], True)
                    if isTheSunDown():
                        switchname(client,["floodtesla","couchwhite","laserball","logo"],"on")
                        switchZigbeeOutlet(client,["w1/OutletBlueLEDBar","w1/OutletAuslageW1"],"ON")
                        switchsonoff(client,["couchred"],"on")
                        switchesphome(client,["subtable"],"on")
                        switchesphome(client,["w1gastherme"],"on")
                        client.publish("action/ceilingscripts/activatescript",'{"script":"redshift","participating":["ceiling2","ceiling3","ceiling4"],"value":0.75,"fadeduration":6000}')
                        # client.publish("action/ceiling1/light",'{"r":400,"b":0,"ww":800,"cw":0,"g":0,"fade":{}}')
                        # client.publish("action/ceiling3/light",'{"r":400,"b":0,"ww":800,"cw":0,"g":0,"fade":{}}')
                    # doppelt hält besser, für die essentiellen dinge
                    switchname(client,["boilerolga","cxleds"],"on")
                else:
                    # everybody left
                    if last_w2_locked_ == 0:
                        last_w2_locked_ = time.time()  #everything locked, start retro-corner-off timer unless w2 was closed earlier
                    client.publish("action/ceilingscripts/activatescript",'{"script":"off"}')
                    client.publish("action/ceilingAll/light",'{"r":0,"b":0,"ww":0,"cw":0,"g":0,"uv":0,"fade":{}}')
                    client.publish("action/ducttape-ledstrip/light",'{"r":0,"b":0,"ww":0,"cw":0,"g":0,"uv":0}') #ducttape light might not listen to ceilingAll
                    switchname(client,["abwasch","couchwhite","laserball","logo","all"],"off")
                    switchZigbeeOutlet(client,["w1/OutletBlueLEDBar","w1/OutletAuslageW1"],"OFF")
                    switchsonoff(client,["couchred","lothrboiler","olgaboiler","mashadecke"],"off")
                    switchesphome(client,["twang","mashacompressor"],"OFF")
                    switchesphome(client,["olgadecke","subtable"],"off")
                    time.sleep(4)
                    switchname(client,["all"],"off")
                    # doppelt hält besser, für die essentiellen dinge
                    client.publish("action/ceilingAll/light",'{"r":0,"b":0,"ww":0,"cw":0,"g":0,"uv":0}')
                    switchname(client,["boilerolga"],"off")
                    switchesphome(client,["w1gastherme"],"off")
            elif last_status["InSpace1"] != dictdata["InSpace1"] and dictdata["Present"] == True:
                ## Presence InSpace1 changed while overall presence remains true
                if dictdata["InSpace1"]:
                    ## Someone came in through the front door
                    switchsonoff(client,["couchred"],"on")
                    switchesphome(client,["w1gastherme"],"on")
                else:
                    ## Everybody left and only people in W2 remain
                    switchsonoff(client,["couchred"],"off")
                    switchesphome(client,["subtable"],"off")
                    switchesphome(client,["w1gastherme"],"off")
                    switchname(client,["basiclightAll"],"off")
            elif last_status["InSpace2"] != dictdata["InSpace2"] and dictdata["Present"] == True:
                if dictdata["InSpace2"]:
                    # switch on stuff in space2 if somebody there
                    last_w2_locked_ = 0 ## 0 means don't switch stuff off
                    switchesphome(client,["twang"],"ON")
                else:
                    ## switch off stuff in space2 if nobody there
                    last_w2_locked_ = time.time()  # w2 locked, start timer to switch of retro-corner
                    switchesphome(client,["twang"],"OFF")
                    client.publish("action/funkbude/light",'{"r":0,"b":0,"ww":0,"cw":0,"g":0,"uv":0,"fade":{}}')
            ### presence stuff that should happen on any presence update anyway
            if dictdata["Present"]:
                # switch single-led green
                if backdoorblue_locked == True and w1_frontdoor_locked == True and  dictdata["InSpace1"] == True and dictdata["InSpace2"] == False:
                    # if all locked but room thinks someone is still there (locked from inside), be yellowish
                    client.publish("action/singleled/light",'{"r":90,"g":100,"b":0}');
                elif dictdata["InSpace1"] == True and dictdata["InSpace2"] == True:
                    # more green if people are present in both rooms
                    client.publish("action/singleled/light",'{"r":10,"g":200,"b":10}');
                else:
                    # less green otherwise
                    client.publish("action/singleled/light",'{"r":0,"g":90,"b":0}');
            else:
                client.publish("action/singleled/light",'{"r":180,"g":0,"b":0}');
        elif topic.endswith("zigbee2mqtt/w1/MashaPIR"):
            if True == dictdata["occupancy"]:
                switchsonoff(client,["mashadecke"],"on")
                last_masha_no_more_movement_ = 0
            else:
                last_masha_no_more_movement_ = time.time()
        elif topic.endswith("zigbee2mqtt/w1/MashaPIR2"):
            if True == dictdata["occupancy"]:
                switchsonoff(client,["mashadecke"],"on")
                last_masha_no_more_movement2_ = 0
            else:
                last_masha_no_more_movement2_ = time.time()
        elif topic.endswith("/boredoombuttonpressed"):
            pass
        elif topic.endswith("realraum/frontdoor/lock"):
            w1_frontdoor_locked = dictdata["Locked"]
        elif topic.endswith("realraum/backdoorcx/lock"):
            backdoorblue_locked = dictdata["Locked"]
        elif topic.endswith("realraum/w2frontdoor/lock"):
            if msg.retain:
                return
            if dictdata["Locked"] == True:
                ## switch on hallwaylight (it should be configured to turn itself of after some seconds)
                switchsonoff(client,["hallwaylight"],"on")
                ## for 30s
                ##scheduleSwitchSonoff(["hallwaylight"],"off",time.time()+40)
        elif topic.endswith("/backdoorcx/ajar") or topic.endswith("/w2frontdoor/ajar"):
            if msg.retain:
                return
            ## switch on hallwaylight (it should be configured to turn itself of after some seconds)
            switchsonoff(client,["hallwaylight"],"on")
            if isTheSunDown() and dictdata["Shut"] == False:
                if topic.endswith("/backdoorcx/ajar"):
                    ## also switch CX light on and leave them on
                    switchname(client,["cxleds"],"on")
        elif topic == topic_tradfri_onoff_lothr:
            if not "click" in dictdata:
                return

            if "on" == dictdata["click"]:
                ### shortclick on
                switchname(client,["floodtesla","ceiling6"],"on")
            elif "off" == dictdata["click"]:
                ### shortclick off
                switchname(client,["floodtesla","ceiling6"],"off")
            elif "brightness_up" == dictdata["click"]:
                ### longpress on has started
                client.publish("action/ceilingscripts/activatescript",json.dumps({"script":"wave","colourlist":[{"r":1000,"g":0,"b":0,"ww":0,"cw":0},{"r":800,"g":0,"b":100,"ww":0,"cw":0},{"r":0,"g":0,"b":300,"ww":0,"cw":0},{"r":0,"g":500,"b":100,"ww":0,"cw":0},{"r":0,"g":800,"b":0,"ww":0,"cw":0},{"r":800,"g":200,"b":0,"ww":0,"cw":0},], "fadeduration":5000}))
            elif "brightness_down" == dictdata["click"]:
                ### longpress off has started
                client.publish("action/ceilingscripts/activatescript",'{"script":"redshift","participating":["ceiling1","ceiling2","ceiling3","ceiling4","ceiling5","ceiling6"],"value":0.7}')
            elif "brightness_stop" == dictdata["click"]:
                ### longpress has stopped
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
    last_status = {}
    unixts_panic_button = None
    unixts_last_movement = 0
    unixts_last_presence = 0
    client = mqtt.Client(client_id=os.path.basename(sys.argv[0]))
    client.on_connect = lambda client, userdata, flags, rc: client.subscribe([
        ("realraum/metaevt/presence", 1),
        ("realraum/metaevt/duskordawn", 1),
        ("realraum/pillar/boredoombuttonpressed", 1),
        ("realraum/+/ajar",1),
        ("realraum/w2frontdoor/lock",1),
        ("realraum/frontdoor/lock",1),
        ("realraum/backdoorcx/lock",1),
        ("zigbee2mqtt/w1/MashaPIR",1),
        ("zigbee2mqtt/w1/MashaPIR2",1),
        (topic_tradfri_onoff_lothr,1),
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
