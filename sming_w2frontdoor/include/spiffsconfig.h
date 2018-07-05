#ifndef INCLUDE_SPIFFSCONFIG_H_
#define INCLUDE_SPIFFSCONFIG_H_

#define MAX_WIFI_SETS 3

class SpiffsConfigStorage
{
public:
	IPAddress ip = IPAddress(192, 168, 127, 242);
	IPAddress netmask = IPAddress(255,255,255,0);
	IPAddress gw = IPAddress(192, 168, 127, 254);
	IPAddress dns[DNS_MAX_SERVERS] = {IPAddress(89,106,211,33),IPAddress(8,8,8,8)};
	String wifi_ssid[MAX_WIFI_SETS]={"","",""};
	String wifi_pass[MAX_WIFI_SETS]={"","",""};
	String mqtt_broker="mqtt.realraum.at";
	String mqtt_clientid="test";
	String mqtt_user;
	String mqtt_pass;
	bool enabledhcp[MAX_WIFI_SETS]={true,true,true};
	uint16_t mqtt_port=1883;  //8883 for ssl
	uint32_t publish_interval=3000;
	String authtoken;

	void load();
	void save();
	bool exist();

	String getWifiSSID() {return wifi_ssid[wifi_settings_idx];}
	String getWifiPASS() {return wifi_pass[wifi_settings_idx];}
	bool getEnableDHCP() {return enabledhcp[wifi_settings_idx];}
	void nextWifi() {wifi_settings_idx++; wifi_settings_idx %= MAX_WIFI_SETS; if (wifi_ssid[wifi_settings_idx].length() == 0) {wifi_settings_idx=0;}}

private:
	uint32_t wifi_settings_idx = 0;

};

extern SpiffsConfigStorage NetConfig;

#endif /* INCLUDE_DEFAULTCONFIG_H_ */
