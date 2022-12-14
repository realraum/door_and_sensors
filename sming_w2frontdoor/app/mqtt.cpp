#include <SmingCore/SmingCore.h>
#include <spiffsconfig.h>
#include "pinconfig.h"
#include "mqtt.h"
#include "application.h"

//////////////////////////////////
/////// MQTT Stuff ///////////////
//////////////////////////////////


Timer procMQTTTimer;
MqttClient *mqtt = nullptr;
bool resend_status_was_requested_ = false;

String getMQTTTopic(String topic3)
{
	return MQTT_TOPIC1+NetConfig.mqtt_clientid+topic3;
}

// Check for MQTT Disconnection
void checkMQTTDisconnect(TcpClient& client, bool flag){

	// Called whenever MQTT connection is failed.
	if (flag == true)
	{
		Serial.println("MQTT Broker Disconnected!!");
	}
	else
	{
		Serial.println("MQTT Broker Unreachable!!");
	}

	// Restart connection attempt after few seconds
	// changes procMQTTTimer callback function
	procMQTTTimer.initializeMs(2 * 1000, startMqttClient).start(); // every 2 seconds
}

void onMessageDelivered(uint16_t msgId, int type) {
	//Serial.printf("Message with id %d and QoS %d was delivered successfully.", msgId, (type==MQTT_MSG_PUBREC? 2: 1));
}

void publishLockMessage(StaticJsonBuffer<1024> &jsonBuffer)
{
	if (nullptr == mqtt)
		return;
	JsonObject& root = jsonBuffer.createObject();
	String message;
	root[JSONKEY_LOCKED] = doorLocked();
	root.printTo(message);
	mqtt->publish(getMQTTTopic(MQTT_TOPIC3_LOCK), message, true); //retain
}

void publishShutMessage(StaticJsonBuffer<1024> &jsonBuffer)
{
	if (nullptr == mqtt)
		return;
	JsonObject& root = jsonBuffer.createObject();
	String message;
	root[JSONKEY_SHUT] = doorShut();
	root.printTo(message);
	mqtt->publish(getMQTTTopic(MQTT_TOPIC3_SHUT), message, true); //retain
}


// Publish our message
void publishMessage()
{
	if (nullptr == mqtt)
		return;
	if (mqtt->getConnectionState() != eTCS_Connected)
		startMqttClient(); // Auto reconnect

	if (checkIfStateChanged() || resend_status_was_requested_)
	{
		StaticJsonBuffer<1024> jsonBuffer;
		publishLockMessage(jsonBuffer);
		publishShutMessage(jsonBuffer);
		resend_status_was_requested_ = false;
	}
}

// Callback for messages, arrived from MQTT server
void onMessageReceived(String topic, String message)
{
	if (topic == MQTT_TOPIC_RESEND_STATUS) {
		resend_status_was_requested_ = true;
	}
}

// Run MQTT client, connect to server, subscribe topics
void startMqttClient()
{
	procMQTTTimer.stop();

	if (nullptr == mqtt)
		mqtt = new MqttClient(NetConfig.mqtt_broker, NetConfig.mqtt_port, onMessageReceived);

	mqtt->setKeepAlive(42);
	mqtt->setPingRepeatTime(21);
	bool usessl=false;
#ifdef ENABLE_SSL
	usessl=true;
	mqtt->addSslOptions(SSL_SERVER_VERIFY_LATER);

	mqtt->setSslClientKeyCert(default_private_key, default_private_key_len,
							  default_certificate, default_certificate_len, NULL, true);
#endif

	//prepare last will
	StaticJsonBuffer<256> jsonBuffer;
	String message;
	JsonObject& root = jsonBuffer.createObject();
	root[JSONKEY_IP] = WifiStation.getIP().toString();
	root[JSONKEY_ONLINE] = false;
	root.printTo(message);
	mqtt->setWill(getMQTTTopic(MQTT_TOPIC3_DEVICEONLINE),message,0,true);

	// Assign a disconnect callback function
	mqtt->setCompleteDelegate(checkMQTTDisconnect);
	// debugf("connecting to to MQTT broker");
	mqtt->connect(NetConfig.mqtt_clientid, NetConfig.mqtt_user, NetConfig.mqtt_pass, true);
	// debugf("connected to MQTT broker");
	mqtt->subscribe(MQTT_TOPIC_RESEND_STATUS);
	//publish fact that we are online
	root[JSONKEY_ONLINE] = true;
	message="";
	root.printTo(message);
	mqtt->publish(getMQTTTopic(MQTT_TOPIC3_DEVICEONLINE),message,true);

	//enable periodic status updates
	procMQTTTimer.initializeMs(NetConfig.publish_interval, publishMessage).start();
}

void stopMqttClient()
{
	// mqtt->unsubscribe(getMQTTTopic(...,true));
	mqtt->setKeepAlive(0);
	mqtt->setPingRepeatTime(0);
	procMQTTTimer.stop();
}

