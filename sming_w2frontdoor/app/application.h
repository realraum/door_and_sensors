#ifndef APPLICATION_H
#define APPLICATION_H

bool doorLocked();
bool doorShut();
bool checkIfStateChanged();
void wifiConnectOk();
void wifiConnectFail();
void connectToWifi();
void ready();
void init();
#endif