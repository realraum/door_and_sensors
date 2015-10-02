#!/usr/bin/python
# -*- coding: utf-8 -*-
from __future__ import with_statement
import os
import os.path
import sys
import logging
import logging.handlers
import time
import threading
import signal
import ConfigParser
import traceback
import shutil
import zmq
import zmq.utils.jsonapi as json
import zmq.ssh.tunnel

logger = logging.getLogger()
logger.setLevel(logging.INFO)
lh_syslog = logging.handlers.SysLogHandler(address="/dev/log",facility=logging.handlers.SysLogHandler.LOG_LOCAL2)
lh_syslog.setFormatter(logging.Formatter('sensorvalues-to-json-zmq.py: %(levelname)s %(message)s'))
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
    self.config_parser.add_section('json')
    self.config_parser.set('json','write_path',"/dev/shm/wget/r3sensors.json")
    self.config_parser.set('json','moveto_path',"/dev/shm/www/r3sensors.json")
    self.config_parser.set('json','backup_path',"/home/guests/realraum.wirdorange.org/r3sensors.json.bak")
    self.config_parser.set('json','backup_every',"50")
    self.config_parser.set('json','limit_list_len',"10000")
    self.config_parser.set('json','updateinterval',"30")
    self.config_parser.add_section('zmq')
    self.config_parser.set('zmq','remote_uri',"tcp://zmqbroker.realraum.at:4244")
    self.config_parser.set('zmq','sshtunnel',"realraum@zmqbroker.realraum.at:22000")
    self.config_parser.set('zmq','sshkeyfile',"/home/guests/realraum.wirdorange.org/id_rsa")
    self.config_parser.set('zmq','subscribe',"TempSensorUpdate IlluminationSensorUpdate DustSensorUpdate RelativeHumiditySensorUpdate MovementSensorUpdate")
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
    finally:
      self.unguardReading()
    return rv


######## r3 ZMQ ############

def sendR3Message(socket, structname, datadict):
    socket.send_multipart([structname, json.dumps(datadict)])

def decodeR3Message(multipart_msg):
    try:
        return (multipart_msg[0], json.loads(multipart_msg[1]))
    except Exception, e:
        logging.debug("decodeR3Message:"+str(e))
        return ("",{})

######## Main ############

def exitHandler(signum, frame):
  logging.info("stopping")
  try:
    zmqsub.close()
    zmqctx.destroy()
  except:
    pass
  sys.exit(0)

time_column_name_="Time"
latest_values_ = {}
sensor_store_ = {}
sensor_cols_num_ = {} #stores number of columns for a sensor not counting Time (x-axis) column. AKA the number of data-rows. Equals highest SensorIndex +1
reset_these_structnames_ = {}

def addEventToTempLastValueStore(structname, msgdata):
    global latest_values_
    sensorindex = int(msgdata["Sensorindex"]) if "Sensorindex" in msgdata else 0
    if not structname in latest_values_:
        latest_values_[structname]=[]
    if not structname in sensor_cols_num_ or sensor_cols_num_[structname] < sensorindex +1:
        sensor_cols_num_[structname] = sensorindex +1
    if len(latest_values_[structname]) < sensor_cols_num_[structname]:
        latest_values_[structname] += [0] * (sensor_cols_num_[structname] - len(latest_values_[structname]))
        expandSensorStoreLists(structname, sensor_cols_num_[structname])
    # store Value in temp last value store:
    try:
        del msgdata["Sensorindex"]
    except:
        pass
    try:
        del msgdata["Ts"]
    except:
        pass
    if len(msgdata) > 0:
        #store first value that is not Sensorindex or Ts into store
        latest_values_[structname][sensorindex] = msgdata.values()[0]
    else:
        #if that value does not exist, (i.e. movementevent), count event occurances
        latest_values_[structname][sensorindex] += 1
        reset_these_structnames_[structname] = True


def cleanTempLastValueOfMovementValues():
    global latest_values_
    for k in reset_these_structnames_.keys():
        latest_values_[k] = [0] * sensor_cols_num_[k]


def expandSensorStoreLists(structname, newlength):
    global sensor_store_
    if not structname in sensor_store_:
        sensor_store_[structname]=[]
    #remove old headings so we can add them again below
    try:
        if sensor_store_[structname][0][0] == time_column_name_:
            sensor_store_[structname].pop(0)
    except:
        pass
    #expand all previous value lists
    newlength_including_time = newlength +1
    sensor_store_[structname] = map(lambda l: l[:newlength_including_time] + ([0] * (newlength_including_time - len(l)))  , sensor_store_[structname])


def addEventsToStore():
    global sensor_store_
    ts = int(time.time())
    for structname in latest_values_.keys():
        if not structname in sensor_store_:
            sensor_store_[structname]=[]

        #if missing, add Header List [Time, 0, 1, 2]
        if len(sensor_store_[structname]) == 0 or len(sensor_store_[structname][0]) < 2 or sensor_store_[structname][0][0] != time_column_name_:
            sensor_store_[structname].insert(0,[time_column_name_] + list(map(lambda n: "Sensor %d"%n,range(0,sensor_cols_num_[structname]))))

        # add values
        try:
            # if latest values are identical, just update timestamp
            if sensor_store_[structname][-1][1:] == latest_values_[structname] and sensor_store_[structname][-1][1:] == sensor_store_[structname][-2][1:]:
                sensor_store_[structname].pop()
        except:
            pass
        sensor_store_[structname].append([ts] + latest_values_[structname])

        #cap list lenght
        if uwscfg.json_limit_list_len:
            if len(sensor_store_[structname]) > uwscfg.json_limit_list_len:
                sensor_store_[structname] = sensor_store_[structname][- uwscfg.json_limit_list_len:]


if __name__ == "__main__":
    #signal.signal(signal.SIGTERM, exitHandler)
    signal.signal(signal.SIGINT, exitHandler)
    signal.signal(signal.SIGQUIT, exitHandler)

    logging.info("%s started" % os.path.basename(sys.argv[0]))

    if len(sys.argv) > 1:
      uwscfg = UWSConfig(sys.argv[1])
    else:
      uwscfg = UWSConfig()

    try:
        with open(uwscfg.json_moveto_path,"rb") as fh:
            sensor_store_ = json.loads(fh.read())
    except Exception, e:
        logging.debug(e)
        try:
            with open(uwscfg.json_backup_path,"rb") as fh:
                sensor_store_ = json.loads(fh.read())
        except Exception, e:
            logging.debug(e)


    for k in set(sensor_store_.keys()).difference(set(uwscfg.zmq_subscribe.split(" "))):
      del sensor_store_[k]  # del old sensordata of sensor we do not subscribe to

    for k in sensor_store_.keys():
      try:
	if len(sensor_store_[k][0]) > 1:
          sensor_cols_num_[k] = len(sensor_store_[k][0]) -1
      except:
        pass

    while True:
      try:
        #Start zmq connection to publish / forward sensor data
        zmqctx = zmq.Context()
        zmqctx.linger = 0
        zmqsub = zmqctx.socket(zmq.SUB)
        for topic in uwscfg.zmq_subscribe.split(" "):
            zmqsub.setsockopt(zmq.SUBSCRIBE, topic)
        if uwscfg.zmq_sshtunnel:
            zmq.ssh.tunnel.tunnel_connection(zmqsub, uwscfg.zmq_remote_uri, uwscfg.zmq_sshtunnel, keyfile=uwscfg.zmq_sshkeyfile)
        else:
            zmqsub.connect(uwscfg.zmq_remote_uri)
        backup_counter = 0
        last_update = int(time.time())

        while True:
            #receive sensor updates
            data = zmqsub.recv_multipart()
            (structname, dictdata) = decodeR3Message(data)
            logging.debug("Got data: " + structname + ":"+ str(dictdata))

            uwscfg.checkConfigUpdates()

            addEventToTempLastValueStore(structname, dictdata)

            logging.debug("lastdata:"+str(latest_values_))
            if int(time.time()) - last_update < int(uwscfg.json_updateinterval):
                continue

            logging.debug("update interval elapsed")
            last_update = int(time.time())

            addEventsToStore()
            cleanTempLastValueOfMovementValues()
            logging.debug("post-cleanMovement lastdata:"+str(latest_values_))

            backup_counter += 1
            # save sensor_store_ to json for apache
            with open(uwscfg.json_write_path,"wb") as fh:
                fh.truncate()
                fh.write(json.dumps(sensor_store_))
            if backup_counter > uwscfg.json_backup_every:
                backup_counter = 0
                shutil.copy(uwscfg.json_write_path, uwscfg.json_backup_path)
            shutil.move(uwscfg.json_write_path, uwscfg.json_moveto_path)

      except Exception, ex:
        logging.error("main: "+str(ex))
        traceback.print_exc(file=sys.stdout)
        try:
          zmqsub.close()
          zmqctx.destroy()
        except:
          pass
        time.sleep(5)


