#include <Arduino.h>
#include <Debug.h>

// Define the pin number for the PIR sensor
const int pirPin = 4;
// Declare and initialize the state variable
int state = 0;
int i = 0;

void setup() {
  pinMode(pirPin, INPUT);  // Set the PIR pin as an input
  Serial.begin(9600);      // Start serial communication with a baud rate of 9600
}

void loop() {
  DEBUG_PRINTLN("Loop iteration: " + String(i++)); // Print the current loop iteration
  state = digitalRead(pirPin);         // Read the state of the PIR sensor
  if (state == HIGH) {                 // If the PIR sensor detects movement (state = HIGH)
    DEBUG_PRINTLN("Somebody here!");  // Print "Somebody here!" to the serial monitor
  } else {
    DEBUG_PRINTLN("Monitoring...");
  }
  delay(1000);
}
