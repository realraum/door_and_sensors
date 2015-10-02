#!/usr/bin/python
# -*- coding: utf-8 -*-
import os
import os.path
import sys
import signal
import zmq.utils.jsonapi as json
import zmq
import traceback
import time
########################

def decodeR3Message(multipart_msg):
    try:
        return (multipart_msg[0], json.loads(multipart_msg[1]))
    except Exception, e:
        logging.debug("decodeR3Message:"+str(e))
        return ("",{})

def exitHandler(signum, frame):
  try:
    zmqsub.close()
    zmqctx.destroy()
  except:
    pass
  sys.exit(0)

signal.signal(signal.SIGINT, exitHandler)
signal.signal(signal.SIGQUIT, exitHandler)

while True:
  try:
    #Start zmq connection to publish / forward sensor data
    zmqctx = zmq.Context()
    zmqctx.linger = 0
    zmqsub = zmqctx.socket(zmq.SUB)
    zmqsub.setsockopt(zmq.SUBSCRIBE, "")
    zmqsub.connect("tcp://zmqbroker.realraum.at:4244")

    while True:

      data = zmqsub.recv_multipart()
      (structname, dictdata) = decodeR3Message(data)
      print "Got data: " + structname + ":"+ str(dictdata)

  except Exception, ex:
    print "main: "+str(ex)
    traceback.print_exc(file=sys.stdout)
    try:
      zmqsub.close()
      zmqctx.destroy()
    except:
      pass
    time.sleep(5)
