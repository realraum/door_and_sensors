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
import subprocess
from collections import defaultdict


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
    self.config_parser.set('mqtt','brokerhost',"mqtt.realraum.at")
    self.config_parser.set('mqtt','brokerport',"1883")
    self.config_parser.add_section('notify')
    self.config_parser.set('notify','emails',"oskar@realraum.at")
    self.config_parser.set('notify','xmpps',"xro@realraum.at")
    self.config_parser.set('notify','smsgroups',"olgacore xro")
    self.config_parser.add_section('sensor')
    self.config_parser.set('sensor','sampleinterval',"15")
    self.config_parser.set('sensor','publishinterval',"59")
    self.config_parser.set('sensor','uri',"http://olgafreezer.realraum.at/")
    self.config_parser.set('sensor','warnunreachablelimit',"6")
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


######## r3 MQTT ############

def sendR3Message(client, topic, datadict, qos=0, retain=False):
    client.publish(topic, json.dumps(datadict), qos, retain)

def sendSMS(groups, message):
    if not isinstance(groups,list):
        groups=[groups]
    print(groups,message)
    smsproc = subprocess.Popen(["/usr/local/bin/send_group_sms.sh"] + groups,stdin=subprocess.PIPE, universal_newlines=True)
    try:
        smsproc.communicate(message, timeout=6)
    except subprocess.TimeoutExpired:
        smsproc.kill()
    print("done")

def sendEmail(groups, message):
    pass

######## Sensor ############

def getJSON(url):
    try:
        r = requests.get(url, timeout=4)
        if r.status_code == 200:
            return r.json()
    except Exception as e:
        print("Error getJSON:", e)
    return {}

unreachable_count = 0
warned_about = defaultdict(bool)
last_publish_ts = defaultdict(int)
def queryTempMonitorAndForward(uwscfg, mqttclient):
    global unreachable_count, warned_about
    jsondict = getJSON(uwscfg.sensor_uri)
    ts = int(time.time())
    if "sensors" in jsondict:
        unreachable_count = 0
        for tsd in jsondict["sensors"]:
            loc=tsd[uwscfg.sensor_locjsonkey]
            temp=float(tsd[uwscfg.sensor_tempjsonkey])
            try:
                warntemp = float(tsd[uwscfg.sensor_warnjsonkey])
            except:
                warntemp = 9999
            if temp > warntemp:
                print("ALARM ALARM %d" % tsd["busid"])
                if not loc in warned_about or warned_about[loc] == False:
                    warned_about[loc]=True
                    msg="Sensor #%d aka %s is @%f degC" % (tsd["busid"], tsd["desc"], tsd["temp"])
                    #send warnings
                    sendSMS(uwscfg.notify_smsgroups.split(" "),msg)
                    sendR3Message(mqttclient, "realraum/olgafreezer/overtemp", {"Location":loc, "Value":temp,"Threshold":warntemp, "Ts":ts}, qos=2, retain=False)
                    sendEmail(uwscfg.notify_emails.split(" "),msg)
            else:
                warned_about[loc]=False

            if ts - last_publish_ts[loc] > int(uwscfg.sensor_publishinterval):
                #print("%s: %f %s" % (loc,tsd[uwscfg.sensor_tempjsonkey], tsd["unit"]))
                sendR3Message(mqttclient, "realraum/olgafreezer/temperature", {"Location":loc, "Value":temp, "Ts":ts}, retain=True)
                last_publish_ts[loc] = ts
    else:
        if unreachable_count >= int(uwscfg.sensor_warnunreachablelimit) and unreachable_count < 2*int(uwscfg.sensor_warnunreachablelimit):
            # if unreachable and before we send error SMS
            # we switch off the power socket that powers the sensor box
            # and then switch it on the next time
            if (unreachable_count - int(uwscfg.sensor_warnunreachablelimit) ) % 2 == 0:
                print("send RF Code for olgatherm OFF (%s)" % time.strftime("%c"))
                sendR3Message(mqttclient, "action/rf433/sendcode3byte", {"Code":list(b"\x00\xa2\x2a"), "Ts":ts}, qos=1, retain=False)
            else:
                print("send RF Code for olgatherm ON (%s)" % time.strftime("%c"))
                sendR3Message(mqttclient, "action/rf433/sendcode3byte", {"Code":list(b"\x00\xa2\x8a"), "Ts":int(time.time())}, qos=1, retain=False)
        elif unreachable_count == 2*int(uwscfg.sensor_warnunreachablelimit):
            timestr = time.strftime("%c")
            sendSMS(["xro"],"OLGA Frige Sensor remains unreachable (%s)" % timestr)
            sendEmail(uwscfg.notify_emails.split(" "),"OLGA Frige Sensor remains unreachable (%s)" % timestr)
            sendR3Message(mqttclient, "realraum/olgafreezer/sensorlost", {"Topic":"realraum/olgafreezer/temperature", "LastSeen":ts - int(uwscfg.sensor_sampleinterval)*int(uwscfg.sensor_warnunreachablelimit), "Ts":ts})
        unreachable_count += 1




############ Main Routine ############

if __name__ == '__main__':

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
    interval_s = float(uwscfg.sensor_sampleinterval)
    assert(interval_s > 1)
    while True:
        try:
            queryTempMonitorAndForward(uwscfg, client)
            starttime = time.time()
            client.loop(timeout=interval_s, max_packets=1)
            remaining_time = interval_s - (time.time() - starttime)
            if remaining_time > 0:
                time.sleep(remaining_time)
        except Exception as e:
            traceback.print_exc()
            print(e)
            break

    client.disconnect()


