/*
  Arduino Uno R4 WiFi - Simple Web Server Template
  
  This example creates a simple web server that:
  - Connects to WiFi
  - Serves a simple HTML page
  - Controls an LED via web interface
  
  Replace YOUR_SSID and YOUR_PASSWORD with your actual WiFi credentials
*/

#include <Arduino.h>
#include <WiFiS3.h>

// WiFi credentials - CHANGE THESE!
const char* ssid = "RobotHub";
const char* password = "robopass";

// Create WiFi server on port 80
WiFiServer server(80);

// LED pin (built-in LED)
const int ledPin = LED_BUILTIN;
bool ledState = false;

void setup() {
  // Initialize serial communication
  Serial.begin(9600);
  while (!Serial) {
    ; // Wait for serial port to connect
  }
  
  // Initialize LED pin
  pinMode(ledPin, OUTPUT);
  digitalWrite(ledPin, LOW);
  
  // Connect to WiFi
  Serial.print("Connecting to ");
  Serial.println(ssid);
  
  WiFi.begin(ssid, password);
  
  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
    Serial.print(".");
  }
  
  Serial.println("");
  Serial.println("WiFi connected!");
  Serial.print("IP address: ");
  Serial.println(WiFi.localIP());
  
  // Start the server
  server.begin();
  Serial.println("Server started");
  Serial.print("Open http://");
  Serial.print(WiFi.localIP());
  Serial.println(" in your browser");
}

void loop() {
  // Listen for incoming clients
  WiFiClient client = server.available();
  
  if (client) {
    Serial.println("New client connected");
    String request = "";
    
    // Read the request
    while (client.connected()) {
      if (client.available()) {
        char c = client.read();
        request += c;
        
        // If we've reached the end of the line (received \n) and the request is complete
        if (c == '\n' && request.endsWith("\r\n\r\n")) {
          break;
        }
      }
    }
    
    // Process the request
    if (request.indexOf("GET /led/on") >= 0) {
      digitalWrite(ledPin, HIGH);
      ledState = true;
      Serial.println("LED turned ON");
    }
    else if (request.indexOf("GET /led/off") >= 0) {
      digitalWrite(ledPin, LOW);
      ledState = false;
      Serial.println("LED turned OFF");
    }
    
    // Send HTTP response
    client.println("HTTP/1.1 200 OK");
    client.println("Content-Type: text/html");
    client.println("Connection: close");
    client.println();
    
    // Send HTML page
    client.println("<!DOCTYPE HTML>");
    client.println("<html>");
    client.println("<head>");
    client.println("<title>Arduino Uno R4 WiFi Web Server</title>");
    client.println("<style>");
    client.println("body { font-family: Arial, sans-serif; margin: 40px; }");
    client.println(".button { display: inline-block; padding: 15px 25px; font-size: 16px; margin: 10px; text-decoration: none; border-radius: 5px; }");
    client.println(".button-on { background-color: #4CAF50; color: white; }");
    client.println(".button-off { background-color: #f44336; color: white; }");
    client.println(".status { font-size: 18px; margin: 20px 0; }");
    client.println("</style>");
    client.println("</head>");
    client.println("<body>");
    client.println("<h1>Arduino Uno R4 WiFi Web Server</h1>");
    client.println("<div class='status'>LED Status: <strong>" + String(ledState ? "ON" : "OFF") + "</strong></div>");
    client.println("<a href='/led/on' class='button button-on'>Turn LED ON</a>");
    client.println("<a href='/led/off' class='button button-off'>Turn LED OFF</a>");
    client.println("<hr>");
    client.println("<p>IP Address: " + WiFi.localIP().toString() + "</p>");
    client.println("<p>RSSI: " + String(WiFi.RSSI()) + " dBm</p>");
    client.println("</body>");
    client.println("</html>");
    
    // Close the connection
    client.stop();
    Serial.println("Client disconnected");
  }
}