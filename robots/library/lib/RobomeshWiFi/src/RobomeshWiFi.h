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
    int getRSSI();

    // Communication methods
    bool tcpSend(const uint8_t* data, size_t length);  // For binary data - returns success
    bool tcpSend(const String& data);                  // For text data - returns success
    size_t tcpReceive(uint8_t* buffer, size_t maxLength);  // Returns actual bytes received
    bool tcpConnect(const char* host, int port);       // Establish TCP connection
    void tcpDisconnect();                              // Close TCP connection gracefully
    bool isTcpConnected();                             // Check if TCP session is active
    void tcpSendHeartbeat();                           // TODO: Send a heartbeat message to keep the connection alive

    void setAuthorizationKey(const char* key);
    
    // TODO: Future encryption methods
    // bool encrypt(const uint8_t* plaintext, size_t length, uint8_t* encrypted, size_t* encryptedLength);
    // bool decrypt(const uint8_t* encrypted, size_t length, uint8_t* plaintext, size_t* plaintextLength);
    // bool tcpSendEncrypted(const uint8_t* data, size_t length);
    // bool tcpSendEncrypted(const String& data);
private:
    WiFiClient client;
    int tcpPort;
    bool _connected;
    char authorizationKey[33]; // 32 characters + null terminator for API key
};

#endif // ROBOMESH_ARDUINO_R4_WIFI_H
