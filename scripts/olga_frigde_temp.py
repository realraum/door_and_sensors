#!/usr/bin/python3
# -*- coding: utf-8 -*-
import os, io
import time
import sys
import logging
import configparser
import traceback
import json
import requests
import threading
import smtplib
import paho.mqtt.client as mqtt


######## Config File Data Class ############

class UWSConfig:
  def __init__(self,configfile=None):
    #Synchronisation
    self.lock=threading.Lock()
    self.finished_reading=threading.Condition(self.lock)
    self.finished_writing=threading.Condition(self.lock)
    self.currently_reading=0
    self.currently_writing=False
    #Config Data
    self.configfile=configfile
    self.config_parser=configparser.ConfigParser()
    self.config_parser.add_section('mqtt')
    self.config_parser.set('mqtt','brokerhost',"mqtt.mgmt.realraum.at")
    self.config_parser.set('mqtt','brokerport',"1883")
    self.config_parser.add_section('notify')
    self.config_parser.set('notify','emails',"oskar@realraum.at")
    self.config_parser.set('notify','xmpps',"xro@realraum.at")
    self.config_parser.add_section('sensor')
    self.config_parser.set('sensor','uri',"http://192.168.33.11/")
    self.config_parser.set('sensor','tempjsonkey',"temp")
    self.config_parser.set('sensor','warnjsonkey',"warnabove")
    self.config_parser.set('sensor','locjsonkey',"desc")
    self.config_mtime=0
    if not self.configfile is None:
      try:
        cf_handle = open(self.configfile,"r")
        cf_handle.close()
      except IOError:
        self.writeConfigFile()
      else:
        self.checkConfigUpdates()

  def guardReading(self):
    with self.lock:
      while self.currently_writing:
        self.finished_writing.wait()
      self.currently_reading+=1

  def unguardReading(self):
    with self.lock:
      self.currently_reading-=1
      self.finished_reading.notifyAll()

  def guardWriting(self):
    with self.lock:
      self.currently_writing=True
      while self.currently_reading > 0:
        self.finished_reading.wait()

  def unguardWriting(self):
    with self.lock:
      self.currently_writing=False
      self.finished_writing.notifyAll()

  def checkConfigUpdates(self):
    global logger
    if self.configfile is None:
      return
    logging.debug("Checking Configfile mtime: "+self.configfile)
    try:
      mtime = os.path.getmtime(self.configfile)
    except (IOError,OSError):
      return
    if self.config_mtime < mtime:
      logging.debug("Reading Configfile")
      self.guardWriting()
      try:
        self.config_parser.read(self.configfile)
        self.config_mtime=os.path.getmtime(self.configfile)
      except (ConfigParser.ParsingError, IOError) as pe_ex:
        logging.error("Error parsing Configfile: "+str(pe_ex))
      self.unguardWriting()
      self.guardReading()
      if self.config_parser.get('debug','enabled') == "True":
        logger.setLevel(logging.DEBUG)
      else:
        logger.setLevel(logging.INFO)
      self.unguardReading()

  def writeConfigFile(self):
    if self.configfile is None:
      return
    logging.debug("Writing Configfile "+self.configfile)
    self.guardReading()
    try:
      cf_handle = open(self.configfile,"w")
      self.config_parser.write(cf_handle)
      cf_handle.close()
      self.config_mtime=os.path.getmtime(self.configfile)
    except IOError as io_ex:
      logging.error("Error writing Configfile: "+str(io_ex))
      self.configfile=None
    self.unguardReading()

  def __getattr__(self, name):
    underscore_pos=name.find('_')
    if underscore_pos < 0:
      raise AttributeError
    rv=None
    self.guardReading()
    try:
      rv = self.config_parser.get(name[0:underscore_pos], name[underscore_pos+1:])
    except (ConfigParser.NoOptionError, ConfigParser.NoSectionError):
      self.unguardReading()
      raise AttributeError
    self.unguardReading()
    return rv


def sendWarningEmail(temp):
    pass

######## r3 MQTT ############

def sendR3Message(client, topic, datadict):
    client.publish(topic, json.dumps(datadict))

######## Sensor ############

def getJSON(url):
    r = requests.get(url)
    if r.status_code == 200:
        return r.json()
    return {}

def queryTempMonitorAndForward(uwscfg):
    rd = getJSON(uwscfg.sensor_uri)
    if uwscfg.sensor_tempjsonkey in rd:
        temp=rd[uwscfg.sensor_tempjsonkey]
        loc=rd[uwscfg.sensor_locjsonkey]
        try:
            warntemp = float(rd[uwscfg.sensor_warnjsonkey])
        except:
            warntemp = -9999
        sendR3Message(mqttclient, "realraum/olgafreezer/temperature", {"Location":loc, "Value":float(temp), "Ts":int(time.time())})
        if temp > warntemp:
            sendR3Message(mqttclient, "realraum/olgafreezer/overtemp", {"Location":loc, "Value":float(temp),"Threshold":warntemp, "Ts":int(time.time())})
            sendWarningEmail(temp)


############ Main Routine ############

logging.info("Olga Fridge Temp Monitor started")

#option and only argument: path to config file
if len(sys.argv) > 1:
  uwscfg = UWSConfig(sys.argv[1])
else:
  uwscfg = UWSConfig()

#Start mqtt connection to publish / forward sensor data
client = mqtt.Client()
client.connect(uwscfg.mqtt_brokerhost, int(uwscfg.mqtt_brokerport), 60)

#listen for sensor data and forward them
while True:
    try:
        queryTempMonitorAndForward(uwscfg)
        client.loop(timeout=5.0, max_packets=1)
    except Exception as e:
        print(e)
        break

client.disconnect()


