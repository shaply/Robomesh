# Tasks

## URGENT

- Add local SQL and maybe mongodb to yoga.
  - Use MySQL for the database so you can practice with it.
- Test the Wifi class for the arduino to see if it works.

## Proposed order of attack

- NEEDS TESTING: Write the RobomeshWifi class in `robots/library/lib/RobomeshWiFi/`
  - Needs `tcpsend` and `tcpreceive`
  - Needs encryption key (but don't need to implement this stuff yet, just include it in the class)
  - Needs to be able to connect to TCP sessions, communicate, disconnect, reconnect
- Create the websocket and website connection framework.
    - When a user clicks on the more for a robot, they should be able to start a websocket connection with the server.
      - This should be optional because maintaining a websocket connection just shouldn't be necessary.
      - Could also pass in SSE ID if needed.
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
- Create the ability to use websockets on frontend and Go server

### Roboserver

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
- Change the layout.svelte files to where the navbar is in the base layout and it renders links, that way, you can move the Notification toast there too.
- Store the auth token as a cookie. Look at notes.md for more info on svelte cookies. Answer is in the website.
- Change the way eSess is fetched because if the eSess is sent before the event listener is added, then it won't set the eSess properly.
- If the backend server is off and the client reaches out, they will fail. But then, if the backend server starts up, the client won't check auth again. Need to add periodic checks.
  - I think the best solution is to just add a loading page where if the person can't connect, it'll redirect to loading page and loading page will keep trying to connect, then when the server is back up, it brings the user back to the page they were on.

## Considerations

- Maybe use some SPA (single page thingy for frontend) to reduce overhead of events

# Notes

- This could be extropolated to treat any device as a robot as long as you create a translation layer, so like you could treat a website as a robot if you create a translation layer, dockerize it so it comes from a "different" ip than localhost, and have it act as an intermediary between the go server the website