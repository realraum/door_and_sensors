#!/usr/bin/python3
# -*- coding: utf-8 -*-
import sys
#import threading
import time
import traceback
import paho.mqtt.client as mqtt
import json
import urllib.request, urllib.parse, urllib.error
import ephem

def isTheSunDown():
    ephemobs=ephem.Observer()
    ephemobs.lat='47.06'
    ephemobs.lon='15.45'
    ephemsun=ephem.Sun()
    ephemsun.compute()
    return ephemobs.date > ephemobs.previous_setting(ephemsun) and ephemobs.date < ephemobs.next_rising(ephemsun)

def decodeR3Message(topic, data):
    try:
        return (topic, json.loads(data.decode("utf-8")))
    except Exception as e:
        #print("decodeR3Message:"+str(e))
        return ("",{})

def touchURL(url):
  try:
    urllib.request.urlcleanup()
    f = urllib.request.urlopen(url)
    rq_response = f.read()
    #print("touchURL: url: "+url)
    f.close()
    return rq_response
  except Exception as e:
    print("touchURL: "+str(e))

def onMqttMessage(client, userdata, msg):
  global last_status, last_user, unixts_panic_button, unixts_last_movement, unixts_last_presence
  if msg.retain:
    return # do not act on retained messages
  (topic, dictdata) = decodeR3Message(msg.topic, msg.payload)
  #print("Got data: " + topic + ":"+ str(dictdata))
  if topic.endswith("/duskordawn") and "HaveSunlight" in dictdata:
    # if people are present and the sun is down, switch on CX Lights
    if last_status:
      if dictdata["HaveSunlight"] == False:
        touchURL("http://licht.realraum.at/cgi-bin/mswitch.cgi?cxleds=1")
      elif dictdata["Event"] == "Sunrise":
        touchURL("http://licht.realraum.at/cgi-bin/mswitch.cgi?cxleds=0")
  elif topic.endswith("/presence") and "Present" in dictdata:
    if dictdata["Present"] and last_status != dictdata["Present"]:
      #someone just arrived
      #power to labortisch so people can switch on the individual lights (and switch off after everybody leaves)
#boiler always on when someone is here
      touchURL("http://licht.realraum.at/cgi-bin/mswitch.cgi?labortisch=1&cxleds=1&boiler=1")
      if isTheSunDown():
        touchURL("http://licht.realraum.at/cgi-bin/mswitch.cgi?ceiling3=1&ceiling4=1&couchred=1&bluebar=1")
      touchURL("http://licht.realraum.at/cgi-bin/mswitch.cgi?boiler=1&labortisch=1") # doppelt hält besser
    last_status=dictdata["Present"]
    if not last_status:
      #everybody left
      touchURL("http://licht.realraum.at/cgi-bin/mswitch.cgi?couchred=0&all=0")
      time.sleep(2)
      touchURL("http://licht.realraum.at/cgi-bin/mswitch.cgi?all=0")
      time.sleep(2)
      touchURL("http://licht.realraum.at/cgi-bin/mswitch.cgi?labortisch=0&boiler=0") # doppelt hält besser

while True:
    last_status=None
    last_user=None
    unixts_panic_button=None
    unixts_last_movement=0
    unixts_last_presence=0
    try:
        client = mqtt.Client()
        client.on_connect = lambda client, userdata, flags, rc: client.subscribe([
            ("realraum/metaevt/presence",1),
            ("realraum/metaevt/duskordawn",1),
            ])
        client.on_message = onMqttMessage
        client.connect("mqtt.realraum.at", 1883, 60)

        # Blocking call that processes network traffic, dispatches callbacks and
        # handles reconnecting.
        # Other loop*() functions are available that give a threaded interface and a
        # manual interface.
        client.loop_forever()

    except Exception as ex:
        print("main: "+str(ex))
        traceback.print_exc(file=sys.stdout)
        try:
          client.disconnect()
        except:
          pass
        time.sleep(5)
