#ifndef USERCONFIG_H
#define USERCONFIG_H

#define LOCK_PIN 5 //D1
 // 2  //D4 10k pullup on D1mini, BUILTIN LED
// 0 //D3 10k pullup on D1mini
// 4  //D2 10k pulldown on D1mini
#define SHUT_PIN 12 //D6

const uint32_t TELNET_PORT_ = 2323;

#define JSONKEY_LOCKED "Locked"
#define JSONKEY_SHUT "Shut"
#define JSONKEY_IP "ip"
#define JSONKEY_ONLINE "online"


const String MQTT_TOPIC1 = "realraum/";
const String MQTT_TOPIC3_LOCK = "/lock";
const String MQTT_TOPIC3_SHUT = "/ajar";
const String MQTT_TOPIC3_DEVICEONLINE = "/online";
const String MQTT_TOPIC_RESEND_STATUS = "action/realraum/resendstatus";


#endif // USERCONFIG_H
