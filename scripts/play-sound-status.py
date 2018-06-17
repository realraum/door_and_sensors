#!/usr/bin/python3
# -*- coding: utf-8 -*-
import os
import os.path
import sys
#import threading
import logging
import logging.handlers
import time
import signal
import subprocess
import types
import configparser
import traceback
import random
import paho.mqtt.client as mqtt
import json

logger = logging.getLogger()
logger.setLevel(logging.INFO)
lh_syslog = logging.handlers.SysLogHandler(
    address="/dev/log",
    facility=logging.handlers.SysLogHandler.LOG_LOCAL2)
lh_syslog.setFormatter(logging.Formatter(
    'play-sound-status.py: %(levelname)s %(message)s'))
logger.addHandler(lh_syslog)
lh_stderr = logging.StreamHandler()
logger.addHandler(lh_stderr)


class UWSConfig:

    def __init__(self, configfile=None):
        self.configfile = configfile
        self.config_parser = configparser.ConfigParser(interpolation=None)
        # make option variable names case sensitive
        self.config_parser.optionxform = str
        self.config_parser.add_section('cmdlog')
        self.config_parser.set('cmdlog', 'cmd', "logger %ARGS%")
        self.config_parser.set('cmdlog', 'timeout', "2.0")
        self.config_parser.set('cmdlog', 'delay', "0.0")
        self.config_parser.set('cmdlog', 'type', "shellcmd")
        self.config_parser.add_section('nothing')
        self.config_parser.set('nothing', 'type', "nothing")
        self.config_parser.add_section('mapping')
        self.config_parser.set('mapping', 'DEFAULT', "nothing")
        self.config_parser.set('mapping', 'PANIC', "nothing")
        self.config_parser.set('mapping', 'ERROR', "nothing")
        self.config_parser.add_section('debug')
        self.config_parser.set('debug', 'enabled', "False")
        self.config_parser.add_section('broker')
        self.config_parser.set('broker', 'host', "mqtt.realraum.at")
        self.config_parser.set('broker', 'port', "1883")
        self.config_parser.add_section('tracker')
        self.config_parser.set(
            'tracker',
            'secs_movement_before_presence_to_launch_event',
            '1')
        self.config_parser.set(
            'tracker',
            'secs_presence_before_movement_to_launch_event',
            '120')
        self.config_mtime = 0
        if not self.configfile is None:
            try:
                cf_handle = open(self.configfile, "r")
                cf_handle.close()
            except IOError:
                self.writeConfigFile()
            else:
                self.checkConfigUpdates()

    def checkConfigUpdates(self):
        global logger
        if self.configfile is None:
            return
        logging.debug("Checking Configfile mtime: " + self.configfile)
        try:
            mtime = os.path.getmtime(self.configfile)
        except (IOError, OSError):
            return
        if self.config_mtime < mtime:
            logging.debug("Reading Configfile")
            try:
                self.config_parser.read(self.configfile)
                self.config_mtime = os.path.getmtime(self.configfile)
            except (configparser.ParsingError, IOError) as pe_ex:
                logging.error("Error parsing Configfile: " + str(pe_ex))
            if self.config_parser.get('debug', 'enabled') == "True":
                logger.setLevel(logging.DEBUG)
            else:
                logger.setLevel(logging.INFO)

    def writeConfigFile(self):
        if self.configfile is None:
            return
        logging.debug("Writing Configfile " + self.configfile)
        try:
            cf_handle = open(self.configfile, "w")
            self.config_parser.write(cf_handle)
            cf_handle.close()
            self.config_mtime = os.path.getmtime(self.configfile)
        except IOError as io_ex:
            logging.error("Error writing Configfile: " + str(io_ex))
            self.configfile = None

    def getValue(self, name):
        underscore_pos = name.find('_')
        if underscore_pos < 0:
            raise AttributeError
        return self.getSectionValue(
            name[
                0:underscore_pos], name[
                underscore_pos + 1:])

    def getSectionValue(self, section, name):
        try:
            return self.config_parser.get(section, name)
        except (configparser.NoOptionError, configparser.NoSectionError):
            return None

    def __getattr__(self, name):
        underscore_pos = name.find('_')
        if underscore_pos < 0:
            raise AttributeError
        try:
            return self.config_parser.get(
                name[
                    0:underscore_pos], name[
                    underscore_pos + 1:])
        except (configparser.NoOptionError, configparser.NoSectionError):
            raise AttributeError


def runRandomAction(action_list, user, args=[]):
    if not isinstance(action_list, list):
        raise ValueError("runRandomAction: action_list must be a list")
    return executeAction(random.choice(action_list), user, args)


def runRemoteCommand(remote_host, remote_shell, user, args=[]):
    global sshp, uwscfg
    sshp = None
    try:
        cmd = "ssh -i /flash/tuer/id_rsa -o PasswordAuthentication=no -o StrictHostKeyChecking=no %RHOST% %RSHELL%"
        cmd = cmd.replace(
            "%RHOST%",
            remote_host).replace(
            "%RSHELL%",
            remote_shell).replace(
            "%ARGS%",
            " ".join(args)).replace(
                "%USER%",
            user)
        logging.debug("runRemoteCommand: Executing: " + cmd)
        sshp = subprocess.Popen(
            cmd.split(" "),
            bufsize=1024,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            shell=False)
        logging.debug(
            "runRemoteCommand: pid %d: running=%d" %
            (sshp.pid, sshp.poll() is None))
        if not sshp.poll() is None:
            logging.error(
                "runRemoteCommand: subprocess %d not started ?, returncode: %d" %
                (sshp.pid, sshp.returncode))
            return False
        return True
    except Exception as ex:
        logging.error("runRemoteCommand: " + str(ex))
        traceback.print_exc(file=sys.stdout)
        if not sshp is None and sshp.poll() is None:
            if sys.hexversion >= 0x020600F0:
                sshp.terminate()
            else:
                subprocess.call(["kill", str(sshp.pid)])
            time.sleep(1.5)
            if sshp.poll() is None:
                logging.error(
                    "runRemoteCommand: subprocess still alive, sending SIGKILL to pid %d" %
                    (sshp.pid))
                if sys.hexversion >= 0x020600F0:
                    sshp.kill()
                else:
                    subprocess.call(["kill", "-9", str(sshp.pid)])
        time.sleep(5)
        return False


def runShellCommand(cmd, ptimeout, stdinput, user, args=[]):
    global uwscfg
    cmd = cmd.replace("%ARGS%", " ".join(args)).replace("%USER%", user)
    if ptimeout is None or float(ptimeout) > 45:
        ptimeout = 45
    else:
        ptimeout = int(float(ptimeout))
    popenTimeout2(cmd, stdinput, ptimeout=ptimeout)


def executeAction(action_name, user, args=[]):
    if action_name is None:
        logging.error("executeAction: action_name is None")
        return False
    action_type = uwscfg.getValue(action_name + "_type")
    if action_type is None:
        logging.error(
            "executeAction: action %s not found or has no type" %
            action_name)
        return False
    action_delay = uwscfg.getValue(action_name + "_delay")
    logging.info(
        "executeAction %s of type %s for user %s with delay %s" %
        (action_name, action_type, user, action_delay))
    if not action_delay is None:
        time.sleep(float(action_delay))

    action_arg = uwscfg.getValue(action_name + "_arg")
    if not action_arg is None:
        args += [action_arg]

    #"registered" actions
    if action_type == "remotecmd":
        return runRemoteCommand(
            uwscfg.getSectionValue(
                action_name, "remote_host"), uwscfg.getSectionValue(
                action_name, "remote_shell"), user=user, args=args)
    elif action_type == "shellcmd":
        return runShellCommand(
            cmd=uwscfg.getSectionValue(
                action_name, "cmd"), ptimeout=uwscfg.getSectionValue(
                action_name, "timeout"), stdinput=uwscfg.getSectionValue(
                action_name, "stdinput"), user=user, args=args)
    elif action_type == "nothing":
        return True
    elif action_type == "random":
        return runRandomAction(
            action_list=uwscfg.getSectionValue(
                action_name,
                "one_of").split(" "),
            user=user,
            args=args)
    else:
        return executeAction(action_type, user=user, args=args)


def playThemeOf(user, fallback_default):
    global uwscfg
    uwscfg.checkConfigUpdates()
    if user is None:
        user = ""
    config = uwscfg.getValue("mapping_" + str(user))
    if config is None:
        config = uwscfg.getValue("mapping_" + str(fallback_default))
    logging.debug("playThemeOf: action for user %s: %s" % (user, config))
    executeAction(config, user, [])


def popenTimeout1(
        cmd,
        pinput,
        returncode_ok=[0],
        ptimeout=20.0,
        pcheckint=0.25):
    logging.debug("popenTimeout1: starting: " + cmd)
    try:
        sppoo = subprocess.Popen(cmd, stdin=subprocess.PIPE, shell=True)
        sppoo.communicate(input=pinput)
        timeout_counter = ptimeout
        while timeout_counter > 0:
            time.sleep(pcheckint)
            timeout_counter -= pcheckint
            if not sppoo.poll() is None:
                logging.debug(
                    "popenTimeout1: subprocess %d finished, returncode: %d" %
                    (sppoo.pid, sppoo.returncode))
                return (sppoo.returncode in returncode_ok)
        # timeout reached
        logging.error(
            "popenTimeout1: subprocess took too long (>%fs), sending SIGTERM to pid %d" %
            (ptimeout, sppoo.pid))
        if sys.hexversion >= 0x020600F0:
            sppoo.terminate()
        else:
            subprocess.call(["kill", str(sppoo.pid)])
        time.sleep(1.0)
        if sppoo.poll() is None:
            logging.error(
                "popenTimeout1: subprocess still alive, sending SIGKILL to pid %d" %
                (sppoo.pid))
            if sys.hexversion >= 0x020600F0:
                sppoo.kill()
            else:
                subprocess.call(["kill", "-9", str(sppoo.pid)])
        return False
    except Exception as e:
        logging.error("popenTimeout1: " + str(e))
        return False


def popenTimeout2(cmd, pinput, returncode_ok=[0], ptimeout=21):
    logging.debug("popenTimeout2: starting: " + cmd)
    try:
        sppoo = subprocess.Popen(cmd, stdin=subprocess.PIPE, shell=True)
        if sys.hexversion >= 0x020600F0:
            old_shandler = signal.signal(
                signal.SIGALRM, lambda sn, sf: sppoo.kill())
        else:
            old_shandler = signal.signal(
                signal.SIGALRM,
                lambda sn,
                sf: os.system(
                    "kill -9 %d" %
                    sppoo.pid))
        signal.alarm(ptimeout)  # schedule alarm
        if not pinput is None:
            sppoo.communicate(input=pinput)
        sppoo.wait()
        signal.alarm(0)  # disable pending alarms
        signal.signal(signal.SIGALRM, old_shandler)
        logging.debug(
            "popenTimeout2: subprocess %d finished, returncode: %d" %
            (sppoo.pid, sppoo.returncode))
        if sppoo.returncode < 0:
            logging.error(
                "popenTimeout2: subprocess took too long (>%ds) and pid %d was killed" %
                (ptimeout, sppoo.pid))
        return (sppoo.returncode in returncode_ok)
    except Exception as e:
        logging.error("popenTimeout2: " + str(e))
        try:
            signal.signal(signal.SIGALRM, old_shandler)
        except:
            pass
        return False


def decodeR3Message(topic, data):
    try:
        return (topic, json.loads(data.decode("utf-8")))
    except Exception as e:
        logging.debug("decodeR3Message:" + str(e))
        return ("", {})


def onMqttMessage(client, userdata, msg):
    global last_status, last_user, unixts_panic_button, unixts_last_movement, unixts_last_presence
    if msg.retain:
        return  # do not act on retained messages

    (topic, dictdata) = decodeR3Message(msg.topic, msg.payload)
    logging.debug("Got data: " + topic + ":" + str(dictdata))

    if topic.endswith("/presence") and "Present" in dictdata:
        if last_status == dictdata["Present"]:
            ##ignore resend of presence or InSpace changes
            return
        unixts_last_presence = time.time()
        last_status = dictdata["Present"]
        unixts_panic_button = None
        if (time.time() - unixts_last_movement) <= float(
                uwscfg.tracker_secs_movement_before_presence_to_launch_event):
            unixts_last_movement = 0
            unixts_last_presence = 0
            if last_status:
                playThemeOf(user=last_user, fallback_default="DEFAULT")
        return
    elif topic.endswith("/cmdevt"):
        last_user = dictdata["Who"]
        return
    elif topic.endswith("/boredoombuttonpressed"):
        playThemeOf(user="PANIC", fallback_default="nothing")
        return
    elif topic.endswith("/movement") or topic.endswith("/ajar"):
        unixts_last_movement = time.time()
        if (time.time() - unixts_last_presence) <= float(
                uwscfg.tracker_secs_presence_before_movement_to_launch_event):
            unixts_last_presence = 0
            unixts_last_movement = 0
            if last_status:
                playThemeOf(user=last_user, fallback_default="DEFAULT")
        return
    elif topic.endswith("/problemevt") and "Severity" in dictdata:
        playThemeOf(user="ERROR", fallback_default="nothing")
        return


def exitHandler(signum, frame):
    logging.info("stopping")
    sys.exit(0)

# signals proapbly don't work because of readline
#signal.signal(signal.SIGTERM, exitHandler)
signal.signal(signal.SIGINT, exitHandler)
signal.signal(signal.SIGQUIT, exitHandler)

logging.info("Door Status Listener 'PlaySound' started")

if len(sys.argv) > 1:
    uwscfg = UWSConfig(sys.argv[1])
else:
    uwscfg = UWSConfig()

while True:
    last_status = None
    last_user = None
    unixts_panic_button = None
    unixts_last_movement = 0
    unixts_last_presence = 0
    try:
        client = mqtt.Client()
        client.on_connect = lambda client, userdata, flags, rc: client.subscribe([
            ("realraum/+/cmdevt", 1),
            ("realraum/metaevt/presence", 1),
            ("realraum/+/boredoombuttonpressed", 1),
            ("realraum/+/ajar", 1),
            ("realraum/pillar/movement", 1),
            ("realraum/+/problemevt", 1),
            ("realraum/", 1),
        ])
        client.on_message = onMqttMessage
        client.connect(uwscfg.broker_host, int(uwscfg.broker_port), 60)

        # Blocking call that processes network traffic, dispatches callbacks and
        # handles reconnecting.
        # Other loop*() functions are available that give a threaded interface and a
        # manual interface.
        client.loop_forever()

    except Exception as ex:
        logging.error("main: " + str(ex))
        traceback.print_exc(file=sys.stdout)
        try:
            client.disconnect()
        except:
            pass
        time.sleep(5)
