#include <user_config.h>
#include <SmingCore/SmingCore.h>
#include <spiffsconfig.h>
#include <pinconfig.h>
#include <SmingCore/Debug.h>
#include "application.h"
#include "telnet.h"
#include "mqtt.h"
#ifdef ENABLE_SSL
	#include <ssl/private_key.h>
	#include <ssl/cert.h>
#endif

uint8_t wifi_fail_count_ = 0;


bool last_locked_=false;
bool last_shut_=false;
bool last_value_known_=false;

bool doorLocked()
{
	return digitalRead(LOCK_PIN) == LOW;
}

bool doorShut()
{
	return digitalRead(SHUT_PIN) == LOW;
}

bool checkIfStateChanged()
{
	bool locked = doorLocked();
	bool shut = doorShut();

	if (last_shut_ != shut || last_locked_ != locked)
	{
		last_locked_ = locked;
		last_shut_ = shut;
		return true;
	}
	return false;
}


void initState()
{
	last_locked_ = doorLocked();
	last_shut_ = doorShut();
}

///////////////////////////////////////
///// WIFI Stuff
///////////////////////////////////////

void configureDNSServer(uint8_t n, IPAddress a)
{
	ip_addr_t adr;
	ip4_addr_set_u32(&adr, (uint32_t) a);
	dns_setserver(n,&adr);
}

void configureWifi()
{
	// ip_addr_t ipdnsr3_;
	// ip_addr_t ipdnsffgraz_;
	// ip4_addr_set_u32(&ipdnsr3_, 0x21D36A59); //89.106.211.33
	// ip4_addr_set_u32(&ipdnsffgraz_,0x0A000B0A); //10.12.0.10
	// dns_setserver(0,&ipdnsr3_);
	// dns_setserver(1,&ipdnsffgraz_);
	for (uint8_t d=0; d<DNS_MAX_SERVERS;d++)
		configureDNSServer(d,NetConfig.dns[d]);
	WifiAccessPoint.enable(false);
	WifiStation.enable(true);
	// Serial.println("clientid: "+NetConfig.mqtt_clientid);
	// Serial.println("SSID: "+NetConfig.getWifiSSID());
	// Serial.println("WifiPass: "+NetConfig.getWifiPASS());
	WifiStation.setHostname(NetConfig.mqtt_clientid+".realraum.at");
	WifiStation.config(NetConfig.getWifiSSID(), NetConfig.getWifiPASS()); // Put you SSID and Password here
	WifiStation.enableDHCP(NetConfig.getEnableDHCP());
	if (!NetConfig.getEnableDHCP())
		WifiStation.setIP(NetConfig.ip,NetConfig.netmask,NetConfig.gw);
}

// Will be called when WiFi station was connected to AP
void wifiConnectOk(IPAddress ip, IPAddress mask, IPAddress gateway)
{
	// debugf("WiFi CONNECTED");
	// Serial.println(ip.toString());
	Serial.println(WifiStation.getIP().toString());
	startTelnetServer();
	startMqttClient();
}


// Will be called when WiFi station timeout was reached
void wifiConnectFail(String ssid, uint8_t ssidLength, uint8_t *bssid, uint8_t reason)
{
	// The different reason codes can be found in user_interface.h. in your SDK.
	debugf("Disconnected from %s. Reason: %d", ssid.c_str(), reason);

	wifi_fail_count_++;
	if (wifi_fail_count_ > 2)
	{
		NetConfig.nextWifi();
		configureWifi();
		wifi_fail_count_ = 0;
	}
}

//////////////////////////////////////
////// Base System Stuff  ////////////
//////////////////////////////////////

void setupPINs() {
	pinMode(LOCK_PIN, INPUT_PULLUP);
	pinMode(SHUT_PIN, INPUT_PULLUP);
	// digitalWrite(SHUT_PIN, LOW);
}

void init()
{
	Serial.begin(SERIAL_BAUD_RATE);
	Serial.systemDebugOutput(true); // Allow debug print to serial
	// Serial.commandProcessing(true);
	// Mount file system, in order to work with files
/*	int slot = rboot_get_current_rom();
#ifndef DISABLE_SPIFFS
	if (slot == 0) {
#ifdef RBOOT_SPIFFS_0
		debugf("trying to mount spiffs at %x, length %d", RBOOT_SPIFFS_0 + 0x40200000, SPIFF_SIZE);
		spiffs_mount_manual(RBOOT_SPIFFS_0 + 0x40200000, SPIFF_SIZE);
#else
		debugf("trying to mount spiffs at %x, length %d", 0x40300000, SPIFF_SIZE);
		spiffs_mount_manual(0x40300000, SPIFF_SIZE);
#endif
	} else {
#ifdef RBOOT_SPIFFS_1
		debugf("trying to mount spiffs at %x, length %d", RBOOT_SPIFFS_1 + 0x40200000, SPIFF_SIZE);
		spiffs_mount_manual(RBOOT_SPIFFS_1 + 0x40200000, SPIFF_SIZE);
#else
		debugf("trying to mount spiffs at %x, length %d", 0x40500000, SPIFF_SIZE);
		spiffs_mount_manual(0x40500000, SPIFF_SIZE);
#endif
	}
#else
	debugf("spiffs disabled");
#endif
*/	//spiffs_mount(); // default auto spiffs mount
	spiffs_mount_manual(0x100000,SPIFF_SIZE);
	setupPINs(); //Init HW
	initState();
	telnetRegisterCmdsWithCommandHandler();
	commandHandler.registerSystemCommands();
	// configure stuff that needs to be done before system is ready
	NetConfig.load(); //loads netsettings from fs
	//INIT WIFI
	configureWifi();
	WifiEvents.onStationGotIP(wifiConnectOk);
	WifiEvents.onStationDisconnect(wifiConnectFail);
	//start
	startMqttClient();
}
