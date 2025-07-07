#include <RobomeshWiFi.h>

RobomeshWiFi::RobomeshWiFi() : _connected(false) {
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
    return "Not connected";
}

String RobomeshWiFi::getMACAddress() {
    uint8_t mac[6];
    WiFi.macAddress(mac);
    String macStr = "";
    for (int i = 0; i < 6; i++) {
        if (mac[i] < 16) macStr += "0";
        macStr += String(mac[i], HEX);
        if (i < 5) macStr += ":";
    }
    macStr.toUpperCase();
    return macStr;
}

int RobomeshWiFi::getRSSI() {
    if (isConnected()) {
        return WiFi.RSSI();
    }
    return 0;
}

void RobomeshWiFi::sendData(const String& data) {
    if (isConnected()) {
        // Implementation depends on your specific protocol/server
        DEBUG_PRINT("Sending data: ");
        DEBUG_PRINTLN(data);
        // Add your actual sending logic here
    } else {
        DEBUG_PRINTLN("Cannot send data: not connected to WiFi");
    }
}

String RobomeshWiFi::receiveData() {
    if (isConnected()) {
        // Implementation depends on your specific protocol/server
        // Add your actual receiving logic here
        return "Example received data";
    } else {
        DEBUG_PRINTLN("Cannot receive data: not connected to WiFi");
        return "";
    }
}
