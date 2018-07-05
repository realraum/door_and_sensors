
#include <SmingCore/SmingCore.h>
#include <spiffsconfig.h>
#include <pinconfig.h>
#include "mqtt.h"
#include "otaupdate.h"
#include "telnet.h"

///////////////////////////////////////
///// Telnet Backup command interface
///////////////////////////////////////

TelnetServer telnetServer;
int16_t auth_num_cmds=0;

const char* telnet_prev_mistakes_msg = "Prevent Mistakes, give auth token";

void telnetCmdNetSettings(String commandLine  ,CommandOutput* commandOutput)
{
	Vector<String> commandToken;
	int numToken = splitString(commandLine, ' ' , commandToken);
	if (auth_num_cmds <= 0)
	{
		commandOutput->println(telnet_prev_mistakes_msg);
		return;
	}
	auth_num_cmds--;
	if (numToken != 3)
	{
		// commandOutput->println("Usage set ip|nm|gw|dhcp|wifissid|wifipass|mqttbroker|mqttport|mqttclientid|mqttuser|mqttpass|fan|sim <value>");
		commandOutput->println("Usage set <field> <val>");
	}
	else if (commandToken[1] == "ip")
	{
		IPAddress newip(commandToken[2]);
		if (!newip.isNull())
			NetConfig.ip = newip;
	}
	else if (commandToken[1] == "nm")
	{
		IPAddress newip(commandToken[2]);
		if (!newip.isNull())
			NetConfig.netmask = newip;
	}
	else if (commandToken[1] == "gw")
	{
		IPAddress newip(commandToken[2]);
		if (!newip.isNull())
			NetConfig.gw = newip;
	}
	else if (commandToken[1] == "wifissid")
	{
		NetConfig.wifi_ssid[0] = commandToken[2];
	}
	else if (commandToken[1] == "wifipass")
	{
		NetConfig.wifi_pass[0] = commandToken[2];
	}
	else if (commandToken[1] == "mqttbroker")
	{
		NetConfig.mqtt_broker = commandToken[2];
	}
	else if (commandToken[1] == "mqttport")
	{
		uint32_t newport = commandToken[2].toInt();
		if (newport > 0 && newport < 65536)
			NetConfig.mqtt_port = newport;
	}
	else if (commandToken[1] == "publishinterval")
	{
    	uint32_t checkintervall = commandToken[2].toInt();
    	commandOutput->printf("%s: '%d'\r\n",commandToken[1].c_str(),checkintervall);
    	if (checkintervall >= 1000 && checkintervall <= 600000)
    	NetConfig.publish_interval = checkintervall;
	}
	else if (commandToken[1] == "mqttclientid")
	{
		commandOutput->printf("%s: '%s'\r\n",commandToken[1].c_str(),commandToken[2].c_str());
	}
	else if (commandToken[1] == "mqttuser")
	{
		commandOutput->printf("%s: '%s'\r\n",commandToken[1].c_str(),commandToken[2].c_str());
	}
	else if (commandToken[1] == "mqttpass")
	{
		commandOutput->printf("%s: '%s'\r\n",commandToken[1].c_str(),commandToken[2].c_str());
	}
	else if (commandToken[1] == "dhcp")
	{
		NetConfig.enabledhcp[0] = commandToken[2] == "1" || commandToken[2] == "true" || commandToken[2] == "yes" || commandToken[2] == "on";
	}
	else if (commandToken[1] == "dns0")
	{
		IPAddress newip(commandToken[2]);
		if (!newip.isNull())
			NetConfig.dns[0] = newip;
	}
	else if (commandToken[1] == "dns1")
	{
		IPAddress newip(commandToken[2]);
		if (!newip.isNull())
			NetConfig.dns[1] = newip;
	} else {
		commandOutput->printf("Invalid subcommand");
	}
}

void telnetCmdPrint(String commandLine  ,CommandOutput* commandOutput)
{
	// commandOutput->println("You are connecting from: " + telnetServer.getRemoteIp().toString() + ":" + String(telnetServer.getRemotePort()));
	// commandOutput->println("== Configuration ==");
	commandOutput->println("WiFi0 SSID: " + NetConfig.wifi_ssid[0] + " actual: "+WifiStation.getSSID());
	commandOutput->println("WiFi0 Pass: " + NetConfig.wifi_pass[0] + " actual: "+WifiStation.getPassword());
	commandOutput->println("Hostname: " + WifiStation.getHostname());
	commandOutput->println("MAC: " + WifiStation.getMAC());
	commandOutput->println("IP: " + NetConfig.ip.toString() + " actual: "+WifiStation.getIP().toString());
	commandOutput->println("NM: " + NetConfig.netmask.toString()+ " actual: "+WifiStation.getNetworkMask().toString());
	commandOutput->println("GW: " + NetConfig.gw.toString()+ " actual: "+WifiStation.getNetworkGateway().toString());
	commandOutput->println("DNS: " + NetConfig.dns[0].toString()+ ", "+NetConfig.dns[1].toString());
	commandOutput->println((NetConfig.enabledhcp[0])?"DHCP: on":"DHCP: off");
	commandOutput->println((WifiStation.isEnabledDHCP())?"actual DHCP: on":"DHCP: off");
	commandOutput->println("MQTT Broker: " + NetConfig.mqtt_broker + ":" + String(NetConfig.mqtt_port));
	commandOutput->println("MQTT ClientID: " + NetConfig.mqtt_clientid);
	commandOutput->println("MQTT Login: " + NetConfig.mqtt_user +"/"+ NetConfig.mqtt_pass);
}

void telnetCmdSave(String commandLine  ,CommandOutput* commandOutput)
{
	if (auth_num_cmds <= 0)
	{
		commandOutput->println(telnet_prev_mistakes_msg);
		return;
	}
	auth_num_cmds--;
	commandOutput->println("OK, saving values...");
	NetConfig.save();
}

void telnetCmdLs(String commandLine  ,CommandOutput* commandOutput)
{
	Vector<String> list = fileList();
	for (int i = 0; i < list.count(); i++)
		commandOutput->println(String(fileGetSize(list[i])) + " " + list[i]);
}

void telnetCmdCatFile(String commandLine  ,CommandOutput* commandOutput)
{
	if (auth_num_cmds <= 0)
	{
		commandOutput->println(telnet_prev_mistakes_msg);
		return;
	}
	auth_num_cmds--;
	Vector<String> commandToken;
	int numToken = splitString(commandLine, ' ' , commandToken);

	if (numToken != 2)
	{
		commandOutput->println("Usage: cat <file>");
		return;
	}
	if (fileExist(commandToken[1]))
	{
		commandOutput->println("Contents of "+commandToken[1]);
		commandOutput->println(fileGetContent(commandToken[1]));
	} else {
		commandOutput->println("File '"+commandToken[1]+"' does not exist");
	}
}

void telnetCmdLoad(String commandLine  ,CommandOutput* commandOutput)
{
	commandOutput->printf("OK, loading values...\r\n");
	NetConfig.load();
}

void telnetCmdReboot(String commandLine  ,CommandOutput* commandOutput)
{
	commandOutput->printf("OK, restarting...\r\n");
	telnetServer.flush();
	telnetServer.close();
	System.restart();
}

void telnetAirUpdate(String commandLine  ,CommandOutput* commandOutput)
{
	Vector<String> commandToken;
	int numToken = splitString(commandLine, ' ' , commandToken);

	if (auth_num_cmds <= 0)
	{
		commandOutput->println(telnet_prev_mistakes_msg);
		return;
	}
	auth_num_cmds--;

	if (2 != numToken )
	{
		commandOutput->println("Usage: update <url to files>");
		return;
	} else
	{
		if (commandToken[1].length() > 0 && commandToken[1].startsWith("http") && commandToken[1].endsWith("/"))
		{
			commandOutput->println("URL OK: "+commandToken[1]);
		} else
		{
			commandOutput->println("Invalid URL: "+commandToken[1]);
			return;
		}
		commandOutput->println("Updating...");
		OtaUpdate(commandToken[1]+"rom0.bin",commandToken[1]+"rom1.bin",commandToken[1]+"spiff_rom.bin");
	}

}

void telnetAuth(String commandLine  ,CommandOutput* commandOutput)
{
	if (commandLine == "auth prevents mistakes "+NetConfig.authtoken)
	{
		auth_num_cmds = 3;
		commandOutput->println("go ahead, use your 3 commands wisely");
	} else {
		auth_num_cmds = 0;
		commandOutput->println("no dice");
	}
}

void startTelnetServer()
{
	telnetServer.listen(TELNET_PORT_);
	telnetServer.enableCommand(true);
	//TODO: use encryption and client authentification
#ifdef ENABLE_SSL
	telnetServer.addSslOptions(SSL_SERVER_VERIFY_LATER);
	telnetServer.setSslClientKeyCert(default_private_key, default_private_key_len,
							  default_certificate, default_certificate_len, NULL, true);
	telnetServer.useSsl = true;
#endif
}

void telnetRegisterCmdsWithCommandHandler()
{
	commandHandler.registerCommand(CommandDelegate("set","Change settings","cG", telnetCmdNetSettings));
	commandHandler.registerCommand(CommandDelegate("save","Save settings","cG", telnetCmdSave));
	commandHandler.registerCommand(CommandDelegate("load","Load settings","cG", telnetCmdLoad));
	commandHandler.registerCommand(CommandDelegate("show","Show settings","cG", telnetCmdPrint));
	commandHandler.registerCommand(CommandDelegate("ls","List files","cG", telnetCmdLs));
	commandHandler.registerCommand(CommandDelegate("cat","Cat file contents","cG", telnetCmdCatFile));
	commandHandler.registerCommand(CommandDelegate("restart","restart ESP8266","sG", telnetCmdReboot));
	commandHandler.registerCommand(CommandDelegate("update","OTA Firmware update","sG", telnetAirUpdate));
	commandHandler.registerCommand(CommandDelegate("auth","auth token","sG", telnetAuth));
}
