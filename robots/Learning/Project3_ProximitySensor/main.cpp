// #include <Arduino.h>
// #include <WiFiS3.h>

// // WiFi credentials
// const char* ssid = "YOUR_WIFI_SSID";
// const char* password = "YOUR_WIFI_PASSWORD";

// // TCP Server details
// const char* server = "192.168.1.100";  // Replace with your server IP
// const int port = 8080;                 // Replace with your server port

// WiFiClient client;

// void setup() {
//   Serial.begin(9600);
  
//   // Wait for serial port to connect
//   while (!Serial) {
//     delay(10);
//   }
  
//   Serial.println("Starting WiFi connection...");
  
//   // Connect to WiFi
//   WiFi.begin(ssid, password);
  
//   while (WiFi.status() != WL_CONNECTED) {
//     delay(500);
//     Serial.print(".");
//   }
  
//   Serial.println("");
//   Serial.println("WiFi connected!");
//   Serial.print("IP address: ");
//   Serial.println(WiFi.localIP());
// }

// void loop() {
//   // Connect to TCP server
//   if (client.connect(server, port)) {
//     Serial.println("Connected to server");
    
//     // Send message to server
//     String message = "Hello from Arduino!";
//     client.println(message);
//     Serial.println("Message sent: " + message);
    
//     // Wait for response
//     unsigned long timeout = millis();
//     while (client.available() == 0) {
//       if (millis() - timeout > 5000) {
//         Serial.println(">>> Client Timeout !");
//         client.stop();
//         return;
//       }
//     }
    
//     // Read response from server
//     while (client.available()) {
//       String response = client.readStringUntil('\r');
//       Serial.println("Server response: " + response);
//     }
    
//     // Close connection
//     client.stop();
//     Serial.println("Connection closed");
    
//   } else {
//     Serial.println("Connection to server failed");
//   }
  
//   // Wait 10 seconds before next connection
//   delay(10000);
// }
