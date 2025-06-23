```c++
WiFiUDP udp;

void setup() {
    ...

    udp.begin(PORT);
}

void loop() {
    if (udp.parsePacket()) {
        datalen = udp.available();

        /* Read packet */
        udp.read(packet, 255);
        packet[datalen] = 0; // Necessary packet bytes might not end

        /* Write packet */
        udp.beginPacket(udp.remoteIP(), udp.remotePort());
        udp.print(...);
        udp.endPacket();
    }
}
```