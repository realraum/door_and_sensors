#!/usr/bin/python3
# -*- coding: utf-8 -*-

import json
import time
import serial
import paho.mqtt.client as mqtt
import traceback
######## r3 ZMQ ############

myclientid_="pillar"
rf433_send_delay_s_ = 0.0
TOPIC_YAMAHA_IR_CMD = "action/yamahastereo/ircmd"
TOPIC_RF433_CMD = "action/rf433/sendcode3byte"
TOPIC_RF433_SETDELAY = "action/rf433/setdelay"
query_sensor_intervall_ = 20

yamaha_arudino_cmds_ = {
    "ymhpoweroff":b":"
    ,"ymhpower":b"."
    ,"ymhpoweron":b":."
    ,"ymhcd":b"1"
    ,"ymhtuner":b"2"
    ,"ymhtape":b"3"
    ,"ymhwdtv":b"4"
    ,"ymhsattv":b"5"
    ,"ymhvcr":b"6"
    ,"ymh7":b"7"
    ,"ymhaux":b"8"
    ,"ymhextdec":b"9"
    ,"ymhtest":b"0"
    ,"ymhtunabcde":b"/"
    ,"ymheffect":b"\\"
    ,"ymhtunplus":b"+"
    ,"ymhtunminus":b"-"
    ,"ymhvolup":b";;;;;;;;;"
    ,"ymhvoldown":b",,,,,,,,,"
    ,"ymhvolmute":b"_"
    ,"ymhmenu":b"#"
    ,"ymhplus":b"\""
    ,"ymhminus":b"!"
    ,"ymhtimelevel":b"="
    ,"ymhprgdown":b"$"
    ,"ymhprgup":b"%"
    ,"ymhsleep":b"("
    ,"ymhp5":b")"
}

def sendR3Message(client, topic, datadict):
    client.publish(topic, json.dumps(datadict))

def decodeR3Payload(payload):
    try:
        return json.loads(payload.decode("utf-8"))
    except Exception as e:
        print("Error decodeR3Payload:"+str(e))
        return {}

def getAndPublishDHT11SensorValues(client):
    pass

def onMQTTMessage(client, userdata, msg):
    #print(msg.topic,msg.payload)
    data = decodeR3Payload(msg.payload)
    if msg.topic == TOPIC_YAMAHA_IR_CMD and "Cmd" in data and isinstance(data["Cmd"], str):
        if data["Cmd"] in yamaha_arudino_cmds_:
            try:
                tty.write(yamaha_arudino_cmds_[data["Cmd"]])
            except Exception as e:
                print("tty write error", e)
    elif msg.topic == TOPIC_RF433_CMD and "Code" in data and isinstance(data["Code"], list) and len(data["Code"]) == 3 and all([x <= 0xff and x >= 0 for x in data["Code"]]):
        time.sleep(rf433_send_delay_s_)
        try:
            tty.write(b">"+bytes(data["Code"]))
        except Exception as e:
            print("tty write error", e)
    elif msg.topic == TOPIC_RF433_SETDELAY and "Location" in data and isinstance(data["Location"],str) and "DelayMs" in data and isinstance(data["DelayMs"], int):
        if data["Location"] == myclientid_:
            rf433_send_delay_s_ == float(data["DelayMs"]) / 1000.0


#Start zmq connection to publish / forward sensor data
def initMQTT():
    client = mqtt.Client(client_id=myclientid_)
    client.connect("mqtt.realraum.at", 1883, 60)
    client.on_message = onMQTTMessage
    client.subscribe([(TOPIC_YAMAHA_IR_CMD,2), (TOPIC_RF433_CMD,2), (TOPIC_RF433_SETDELAY,2)])
    return client

#Initialize TTY interface
def initTTY():
    tty = serial.Serial(port='/dev/ttyUSB0', baudrate=9600,timeout=1)
    tty.flushInput()
    tty.flushOutput()
    return tty

def handle_arduino_output(client,tty):
    while tty.inWaiting() > 0:
        sensordata = tty.readline()
        if sensordata is None or len(sensordata) <= 2:
            continue
        sensordata = sensordata[:-2]
        if sensordata == b'PanicButton':
            sendR3Message(client,"realraum/"+myclientid_+"/boredoombuttonpressed",{"Ts":int(time.time())})
        elif sensordata == b'movement':
            sendR3Message(client, "realraum/"+myclientid_+"/movement", {"Sensorindex":0, "Ts":int(time.time())})
        elif sensordata[:9] == b"Sensor ?:":
            light = int(sensordata[9:])
            sendR3Message(client, "realraum/"+myclientid_+"/illumination", {"Location":"LoTHR", "Value":light, "Ts":int(time.time())})
        elif sensordata[:9] == b"Sensor *:":
            temp = float(sensordata[9:])
            sendR3Message(client, "realraum/"+myclientid_+"/temperature", {"Location":"LoTHR", "Value":temp, "Ts":int(time.time())})
        elif sensordata == b'OK':
            continue

if __name__ == '__main__':
    while True:
        tty = None
        client = None
        last_get_sensor_data_ts = time.time()
        try:
            tty = initTTY()
            ## if e.g. ttyUSB0 is not available, then code must not reach this line !!
            ## otherwise we continously try to establish a zmq connection just to close it again
            client = initMQTT()
            #client.start_loop()
            while True:
                if time.time() - last_get_sensor_data_ts > query_sensor_intervall_:
                    getAndPublishDHT11SensorValues(client)
                    tty.write(b'?')  # query illumination sensor
                    #tty.write(b'*')  # query temp sensor
                    last_get_sensor_data_ts = time.time()
                handle_arduino_output(client,tty)
                client.loop()

        except Exception as e:
            traceback.print_exc()
        finally:
            if tty:
                tty.close()
            if isinstance(client,mqtt.Client):
                #client_stop_loop()
                client.disconnect()
            time.sleep(5)
