### Disclaimer
Some of the components on TinkerCad have pins that differ in responsibility from the actual components.

Designs can be found in `Component_Circuits/`

# Notes on Components

## PIR Motion Sensor
- The pins of this are as follows: GND, OUTPUT, VCC from left to right where the left is the pin that aligns with the yellow thingy in the corner
- It detects motion by things with heat (so this includes living things), but there is a limiter on it where the OUTPUT pin can be high for only a certain amount of time, then it is low for a little bit, and then it can be high again.
```c++
#include <Arduino.h>

// Define the pin number for the PIR sensor
const int pirPin = 4;
// Declare and initialize the state variable
int state = 0;

void setup() {
  pinMode(pirPin, INPUT);  // Set the PIR pin as an input
  Serial.begin(9600);      // Start serial communication with a baud rate of 9600
}

void loop() {
  state = digitalRead(pirPin);         // Read the state of the PIR sensor
  if (state == HIGH) {                 // If the PIR sensor detects movement (state = HIGH)
    Serial.println("Somebody here!");  // Print "Somebody here!" to the serial monitor
  } else {
    Serial.println("Monitoring...");
  }
  delay(1000);
}
```