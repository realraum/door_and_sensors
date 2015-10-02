#!/usr/bin/python
# -*- coding: utf-8 -*-
from __future__ import with_statement
import os
import os.path
import sys
import threading
import logging
import logging.handlers
import time
import signal
import re
import subprocess
import ConfigParser
import traceback
import json
import zmq

logger = logging.getLogger()
logger.setLevel(logging.INFO)
lh_syslog = logging.handlers.SysLogHandler(address="/dev/log",facility=logging.handlers.SysLogHandler.LOG_LOCAL2)
lh_syslog.setFormatter(logging.Formatter('bridge-old-sensors.py: %(levelname)s %(message)s'))
logger.addHandler(lh_syslog)
lh_stderr = logging.StreamHandler()
logger.addHandler(lh_stderr)

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
    self.config_parser=ConfigParser.ConfigParser()
    self.config_parser.add_section('sensors')
    self.config_parser.set('sensors','remote_cmd',"ssh -i /flash/tuer/id_rsa -o PasswordAuthentication=no -o StrictHostKeyChecking=no %RHOST% %RSHELL% %RSOCKET%")
    self.config_parser.set('sensors','remote_host',"root@slug.realraum.at")
    self.config_parser.set('sensors','remote_socket',"/var/run/powersensordaemon/cmd.sock")
    self.config_parser.set('sensors','remote_shell',"usocket")
    self.config_parser.add_section('debug')
    self.config_parser.set('debug','enabled',"False")
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
      except (ConfigParser.ParsingError, IOError), pe_ex:
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
    except IOError, io_ex:
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


######## r3 ZMQ ############

def sendR3Message(socket, structname, datadict):
    socket.send_multipart([structname, json.dumps(datadict)])

######## Sensor Bridge ############
tracksensor_running=True
def trackSensorStatus(uwscfg, zmqsocket):
  global sshp, tracksensor_running
  RE_TEMP = re.compile(r'temp(\d): (\d+\.\d+)')
  RE_PHOTO = re.compile(r'photo(\d): [^0-9]*?(\d+)',re.I)
  RE_MOVEMENT = re.compile(r'movement',re.I)
  RE_BUTTON = re.compile(r'button\d?|PanicButton',re.I)
  RE_ERROR = re.compile(r'Error: (.+)',re.I)
  while tracksensor_running:
    uwscfg.checkConfigUpdates()
    sshp = None
    try:
      cmd = uwscfg.sensors_remote_cmd.replace("%RHOST%",uwscfg.sensors_remote_host).replace("%RSHELL%",uwscfg.sensors_remote_shell).replace("%RSOCKET%",uwscfg.sensors_remote_socket).split(" ")
      logging.debug("trackSensorStatus: Executing: "+" ".join(cmd))
      sshp = subprocess.Popen(cmd, bufsize=1024, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, shell=False)
      logging.debug("trackSensorStatus: pid %d: running=%d" % (sshp.pid,sshp.poll() is None))
      if not sshp.poll() is None:
        raise Exception("trackSensorStatus: subprocess %d not started ?, returncode: %d" % (sshp.pid,sshp.returncode))
      #sshp.stdin.write("listen movement\nlisten button\nlisten sensor\n")
      time.sleep(5) #if we send listen bevor usocket is running, we will never get output
      #sshp.stdin.write("listen all\n")
      logging.debug("trackSensorStatus: send: listen movement, etc")
      sshp.stdin.write("listen movement\n")
      sshp.stdin.write("listen button\n")
      sshp.stdin.write("listen sensor\n")
      #sshp.stdin.write("sample temp0\n")
      sshp.stdin.flush()
      while tracksensor_running:
        if not sshp.poll() is None:
          raise Exception("trackSensorStatus: subprocess %d finished, returncode: %d" % (sshp.pid,sshp.returncode))
        line = sshp.stdout.readline()
        if len(line) < 1:
          raise Exception("EOF on Subprocess, daemon seems to have quit, returncode: %d",sshp.returncode)
        logging.debug("trackSensorStatus: Got Line: " + line)
        m = RE_BUTTON.match(line)
        if not m is None:
          sendR3Message(zmqsocket, "BoreDoomButtonPressEvent", {"Ts":int(time.time())})
          continue
        m = RE_MOVEMENT.match(line)
        if not m is None:
          sendR3Message(zmqsocket, "MovementSensorUpdate", {"Sensorindex":0, "Ts":int(time.time())})
          continue
        m = RE_TEMP.match(line)
        if not m is None:
          sendR3Message(zmqsocket, "TempSensorUpdate", {"Sensorindex":int(m.group(1)), "Value":float(m.group(2)), "Ts":int(time.time())})
          continue
        m = RE_PHOTO.match(line)
        if not m is None:
          sendR3Message(zmqsocket, "IlluminationSensorUpdate", {"Sensorindex":int(m.group(1)), "Value":int(m.group(2)), "Ts":int(time.time())})
          continue
        m = RE_ERROR.match(line)
        if not m is None:
          logging.error("trackSensorStatus: got: "+line) 
    except Exception, ex:
      logging.error("trackSensorStatus: "+str(ex)) 
      traceback.print_exc(file=sys.stdout)
      if not sshp is None and sshp.poll() is None:
        if sys.hexversion >= 0x020600F0:
          sshp.terminate()
        else:
          subprocess.call(["kill",str(sshp.pid)])
        time.sleep(1.5)
        if sshp.poll() is None:
          logging.error("trackSensorStatus: subprocess still alive, sending SIGKILL to pid %d" % (sshp.pid))
          if sys.hexversion >= 0x020600F0:
            sshp.kill()
          else:
            subprocess.call(["kill","-9",str(sshp.pid)])
      time.sleep(5)  

 ############ Main Routine ############

def exitHandler(signum, frame):
  global tracksensor_running, sshp
  logging.info("Bridge stopping")
  tracksensor_running=False
  try:
    if sys.hexversion >= 0x020600F0:
      sshp.terminate()
    else:
      subprocess.call(["kill",str(sshp.pid)])
  except:
    pass
  time.sleep(0.1)
  try:
    zmqpub.close()
    zmqctx.destroy()
  except:
    pass
  sys.exit(0)
  
#signals proapbly don't work because of readline
#signal.signal(signal.SIGTERM, exitHandler)
signal.signal(signal.SIGINT, exitHandler)
signal.signal(signal.SIGQUIT, exitHandler)

logging.info("Sensor Bridge started")

#option and only argument: path to config file
if len(sys.argv) > 1:
  uwscfg = UWSConfig(sys.argv[1])
else:
  uwscfg = UWSConfig()

#Start zmq connection to publish / forward sensor data
zmqctx = zmq.Context()
zmqctx.linger = 0
zmqpub = zmqctx.socket(zmq.PUB)
zmqpub.connect("tcp://zmqbroker.realraum.at:4243")

#listen for sensor data and forward them
trackSensorStatus(uwscfg, zmqpub)

zmqpub.close()
zmqctx.destroy()


