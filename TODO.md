# Tasks

## URGENT

- Create the EventBus, so create the publisher subscriber interface for both to use
  - The websockets will now just subscribe to the EventBus
  - The robot manager can store a `SafeSet` with each robot that is trying to register so other servers can easily get the enqueue robots and the robot manager can spawn a Go routine for each robot that's trying to register that watches for the event `deviceID.register` success or failure and then removes it from the enqueue set.
  - Need to change the Register flow to tell the robot that it is enqueue so it knows its REGISTER request arrived
  - Subscriber is an interface that has a Handle(Event) method
  - The backing publisher data structure is a `map[string]{map[subscriber]*node, *node}`
  - `node` has the `prev`, `next`, `subscriber`
  - Then you can call eventbus.subscribe(eventType, subscriber)
  - Then publishers can call eventbus.publish(event)
  - Can implement a base struct for event bus and 
  - Can use this and extend it to general robots so video robots can easily transfer video
  - The events should be named like `deviceid.event_name` and the general events will be `robot_manager.event_name`
  - Use the safe doubly linked list from proximity chat and a map (not the main overarching subscriptions map, just a map to hold location of client) to handle subcribing and unsubscribing
  - Each WSClient should have their own Go routine that handles a message queue and sending a message to all subscribers essentially just adds the message on the message queue to prevent blocking

## Proposed order of attack

- NEEDS TESTING: Write the RobomeshWifi class in `robots/library/lib/RobomeshWiFi/`
  - Needs `tcpsend` and `tcpreceive`
  - Needs encryption key (but don't need to implement this stuff yet, just include it in the class)
  - Needs to be able to connect to TCP sessions, communicate, disconnect, reconnect
- Create the register robot web flow where the user chooses to if the robot should be stored and registered. Use websockets.
  - The robot manager can store a `SafeSet` with each robot that is trying to register so other servers can easily get the enqueue robots and the robot manager can spawn a Go routine for each robot that's trying to register that watches for the event `deviceID.register` success or failure and then removes it from the enqueue set.
  - Need to notify robot of two stages `ENQUEUED` and `OK`. The `ENQUEUED` is to let the robot know that the server got the request and is waiting for verification. On any problems, write `ERROR <ERROR REASON>`
  - Websockets subscribe to the event `robot_manager.register`
  - ALSO, change the robot information card page to use websockets as well.
  - The register robot flow should be a method of the RobomeshWifi class
  - Robot with device ID and a randomly generated unique encryption key should be stored after successful registration
    - Use MongoDB because need a NoSQL for less restricted robot storing, maybe other db but Mongo seems like best for now
    - Perhaps also store a local SQL database with just deviceID, key for quick robot recognition
- Code the encrypted messaging, Symmetric encryption, with the key
  - Complete the unique encryption key flow between robot and server
  - Make sure the encryption is relatively light weight
  - Create the `EREGISTER deviceID encrypted_registration` for the Go/TCP server
    - On failure, make it fall back to the `REGISTER` and where the user chooses if the robot should be allowed on the frontend
    - The server should look up the encryption key and decrypt the `encrypted_registration` and it should result in the `deviceID`
    - If the SQL database was implemented, use this instead of the MongoDB or with the MongoDB like local first then Mongo
- Create the proximity sensor robot
  - When a human or living thing is detected, it sends a message to the server, which stores the time of the detection in a database, then the frontend website displays a graph of the information.
  - Implement a quick action like a connection ping check
  - Implement a Terminal server command for the robot like connected status or list latest n detections

## Ambiguous next steps

### Flows
- Create Login register flow where user can login to the website
  - Will need to consider if the robots are stored with each user so they can login to other machines and have the same robots already registered to them
  - However, robots should also be registered to device
  - Maybe, each device comes with its own account and when changing devices, you get a file that is the export of all registered devices with it and you can upload them to the other device
- Create a robot that just lists the time using the LED thing and it will just display the hour number
  - If have another arduino, test the proximity robot and the clock robot in tandem

### Roboserver

- Create the HTTP GET and POST for the robot communication
- Check how TCP connection works, check if communication cuts off or if Go keeps the connection up
- Change REGISTER for new robots where you need to go on the website to register a new robot to connect it and it logs it to a database. This could also pave way for using a encryption password. This could also then be logged to user accounts and the device.
  - This should edit the RobotManager.Register method
  - Once authentication token is used, change the AddRobot with same deviceID and different ip scenario to use it.
- Add more terminal commands
- Tests for various methods
- Add firewall to TCP server to only accept connections from devices connected to the network
- Create UDP server
  - Make a logging instance of it that logs any incoming messages from robots that want to log for easy logging
- Documentation
- Create validation methods for various fields
- Create the MQTT server stuff
- Change the proximity sensor robot to send detections to the MQTT server
- Push the arduino libraries to github so don't need to include a directory path
- Create a TCP pingpong easy method, so `void tcpSendHeartbeat()` in RobomeshWifi and it should just be a simple to call keep alive method or is connected method for the user

### Robots
- Test TCP send stuff
- Implement the server restart communication handling
  - Basically, when a message fails to send because the server is unreachable, enter a state of waiting for the server to come back to life or something like that and when the server comes back, reregister, then continue as normal

### Frontend_app
- Create the global websocket