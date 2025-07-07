#ifndef ROBOMESH_ARDUINO_R4_WIFI_H
#define ROBOMESH_ARDUINO_R4_WIFI_H

#include <Arduino.h>
#include <WiFiS3.h>
#include <Debug.h>

class RobomeshWiFi {
public:
    RobomeshWiFi(int tcpPort = 80);
    
    // WiFi connection methods
    bool begin(const char* ssid, const char* password);
    bool isConnected();
    void disconnect();
    
    // Network information
    String getIPAddress();
    String getMACAddress();
    int getRSSI();
    
    // Example methods for your specific use case
    void sendData(const String& data);
    String receiveData();

private:
    WiFiClient client;
    int tcpPort;
    bool _connected;
};

#endif // ROBOMESH_ARDUINO_R4_WIFI_H
