#!/usr/bin/python
# -*- coding: utf-8 -*-
from __future__ import with_statement
import zmq.utils.jsonapi as json
import zmq
import time

######## r3 ZMQ ############

def sendR3Message(socket, structname, datadict):
    socket.send_multipart([structname, json.dumps(datadict)])

#Start zmq connection to publish / forward sensor data
zmqctx = zmq.Context()
zmqctx.linger = 0
zmqpub = zmqctx.socket(zmq.PUB)
zmqpub.connect("tcp://torwaechter.realraum.at:4243")

time.sleep(5)
#listen for sensor data and forward them
sendR3Message(zmqpub,"PresenceUpdate",{"Present":True,"Ts":int(time.time())})

zmqpub.close()
zmqctx.destroy()


