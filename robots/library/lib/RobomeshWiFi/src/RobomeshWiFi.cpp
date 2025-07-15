#include <RobomeshWiFi.h>

RobomeshWiFi::RobomeshWiFi(int tcpPort = 80) : _connected(false), tcpPort(tcpPort) {
    // Constructor
}

bool RobomeshWiFi::begin(const char* ssid, const char* password) {
    DEBUG_PRINT("Connecting to WiFi: ");
    DEBUG_PRINTLN(ssid);
    
    WiFi.begin(ssid, password);
    
    // Wait for connection
    int attempts = 0;
    while (WiFi.status() != WL_CONNECTED && attempts < 20) {
        delay(500);
        DEBUG_PRINT(".");
        attempts++;
    }
    
    if (WiFi.status() == WL_CONNECTED) {
        _connected = true;
        DEBUG_PRINTLN();
        DEBUG_PRINTLN("WiFi connected!");
        DEBUG_PRINT("IP address: ");
        DEBUG_PRINTLN(WiFi.localIP());
        return true;
    } else {
        _connected = false;
        DEBUG_PRINTLN();
        DEBUG_PRINTLN("WiFi connection failed!");
        return false;
    }
}

bool RobomeshWiFi::isConnected() {
    return WiFi.status() == WL_CONNECTED && _connected;
}

void RobomeshWiFi::disconnect() {
    WiFi.disconnect();
    _connected = false;
    DEBUG_PRINTLN("WiFi disconnected");
}

String RobomeshWiFi::getIPAddress() {
    if (isConnected()) {
        return WiFi.localIP().toString();
    }
    return "";
}

int RobomeshWiFi::getRSSI() {
    if (isConnected()) {
        return WiFi.RSSI();
    }
    return 0;
}

void RobomeshWiFi::setAuthorizationKey(const char* key) {
    strncpy(authorizationKey, key, sizeof(authorizationKey) - 1);
    authorizationKey[sizeof(authorizationKey) - 1] = '\0'; // Ensure null termination
}

bool RobomeshWiFi::tcpSend(const uint8_t* data, size_t length) {
    if (!isConnected() || !client.connected()) {
        DEBUG_PRINTLN("Cannot send data: not connected to WiFi or TCP");
        return false;
    }
    
    size_t bytesSent = client.write(data, length);
    DEBUG_PRINT("Sent ");
    DEBUG_PRINT_DEC(bytesSent);
    DEBUG_PRINT(" of ");
    DEBUG_PRINT_DEC(length);
    DEBUG_PRINTLN(" bytes");
    
    return bytesSent == length;  // Return true only if all bytes were sent
}

bool RobomeshWiFi::tcpSend(const String& data) {
    if (!isConnected() || !client.connected()) {
        DEBUG_PRINTLN("Cannot send data: not connected to WiFi or TCP");
        return false;
    }
    
    size_t bytesSent = client.print(data);
    DEBUG_PRINT("Sent ");
    DEBUG_PRINT_DEC(bytesSent);
    DEBUG_PRINT(" of ");
    DEBUG_PRINT_DEC(data.length());
    DEBUG_PRINTLN(" bytes");

    return bytesSent == data.length();
}

size_t RobomeshWiFi::tcpReceive(uint8_t* buffer, size_t maxLength) {
    if (!isConnected() || !client.connected()) {
        DEBUG_PRINTLN("Cannot receive data: not connected to WiFi or TCP");
        return 0;
    }
    
    size_t bytesReceived = 0;
    if (client.available()) {
        bytesReceived = client.readBytes(buffer, maxLength);
        DEBUG_PRINT("Received ");
        DEBUG_PRINT_DEC(bytesReceived);
        DEBUG_PRINTLN(" bytes");
    }
    
    return bytesReceived;
}
