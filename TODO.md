# Tasks

### Proposed order of attack

- Write the RobomeshWifi class in `robots/library/lib/RobomeshWiFi/`
  - Needs `tcpsend` and `tcpreceive`
  - Needs encryption key (but don't need to implement this stuff yet, just include it in the class)
  - Needs to be able to connect to TCP sessions, communicate, disconnect, reconnect
- Create the register robot web flow where the user chooses to if the robot should be stored and registered
  - Robot with device ID and a randomly generated unique encryption key should be stored after successful registration
    - Use MongoDB because need a NoSQL for less restricted robot storing, maybe other db but Mongo seems like best for now
    - Perhaps also store a local SQL database with just deviceID, key for quick robot recognition
- Code the encrypted messaging, Symmetric encryption, with the key
  - Make sure the encryption is relatively light weight
  - Create the `EREGISTER encrypted_registration` for the Go/TCP server
    - On failure, make it fall back to the `REGISTER` and where the user chooses if the robot should be allowed on the frontend
    - If the SQL database was implemented, use this instead of the MongoDB or with the MongoDB like local first then Mongo
- Create the proximity sensor robot
  - When a human or living thing is detected, it sends a message to the server, which stores the time of the detection in a database, then the frontend website displays a graph of the information.
  - Implement a quick action like a connection ping check
  - Implement a Terminal server command for the robot like connected status or list latest n detections

### Ambiguous next steps

- Create the MQTT server stuff
- Change the proximity sensor robot to send detections to the MQTT server
- Push the libraries to github so don't need to include a directory path
- Create a robot that just lists the time using the LED thing and it will just display the hour number
  - If have another arduino, test the proximity robot and the clock robot in tandem
