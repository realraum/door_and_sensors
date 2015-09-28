
#!/usr/bin/python
# -*- coding: utf-8 -*-
from __future__ import with_statement
import zmq.utils.jsonapi as json
import zmq
import time
import serial
######## r3 ZMQ ############

def sendR3Message(socket, structname, datadict):
    print socket.send_multipart([structname, json.dumps(datadict)])

#Start zmq connection to publish / forward sensor data
def initZMQ():
    zmqctx = zmq.Context()
    zmqctx.linger = 0
    zmqpub = zmqctx.socket(zmq.PUB)
    zmqpub.connect("tcp://torwaechter.realraum.at:4243")
    return zmqpub,zmqctx
    
#Initialize TTY interface
def initTTY():
    tty = serial.Serial(port='/dev/ttyUSB0', baudrate=9600,timeout=30)
    tty.flushInput()
    tty.flushOutput()
    return tty
    
#listen for sensor data and forward them    
def handle_sensors(zmqpub,tty):
    sensordata = tty.readline()
    sensordata = sensordata[:-2] 
    if sensordata == 'PanicButton':
        sendR3Message(zmqpub,"BoreDoomButtonPressEvent",{"Ts":int(time.time())})
    
    elif sensordata == 'movement':
        sendR3Message(zmqpub, "MovementSensorUpdate", {"Sensorindex":0, "Ts":int(time.time())})

    tty.write('*')
    sensordata = tty.readline()
    sensordata = sensordata[:-2]
    temp = float(sensordata[9:])
    sendR3Message(zmqpub, "TempSensorUpdate", {"Sensorindex":0, "Value":temp, "Ts":int(time.time())})

    tty.write('?')
    sensordata = tty.readline()
    sensordata = sensordata[:-2]
    light = int(sensordata[9:])
    sendR3Message(zmqpub, "IlluminationSensorUpdate", {"Sensorindex":0, "Value":light, "Ts":int(time.time())})

if __name__ == '__main__':
    while True:
        tty = None
        zmqpub = None
        zmqctx = None
        try:
            tty = initTTY()
            ## if e.g. ttyUSB0 is not available, then code must not reach this line !!
            ## otherwise we continously try to establish a zmq connection just to close it again
            zmqpub,zmqctx = initZMQ()
            while True:
                handle_sensors(zmqpub,tty)
        except:
            pass
        finally:
            if isinstance(tty,file):
                tty.close()
            if isinstance(zmqpub,zmq.Socket):
                zmqpub.close()
            if isinstance(zmqctx,zmq.Context):
                zmqctx.destroy()
            time.sleep(5)
