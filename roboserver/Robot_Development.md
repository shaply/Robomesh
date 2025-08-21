# Intro

This guide is to help add a robot to the robomesh program. The parts covered in this guide are 
- Understanding the general flow of the robot handler
- Knowing which methods to write
- The base robot helper types
- The event bus
- Integrating with the frontend
This should make developing a robot and adding it to the robomesh easier.

## Folder Deep Dive

First, let's discuss the folders.

- `build/`: This is where the Go program is compiled to. The compiled program is according to the docker build file.
- `database/`: This stores the database managing files. It will provide libraries to help you easily just store and retrieve from databases.
- `http_server/`: This stores the http server contents. The `http_events/` stores the manager and things for the SSE.
...

## General Flow of the Robot Handler

First, let's talk about how a robot is registered to the robomesh. The following are simplified messages as described by `tcp_server/robot_communication.md`.

The robot registers itself to the robomesh by sending a register message. Then, the tcp server will use one of the robot manager's methods to publish an event to the `REGISTERING_ROBOT_EVENT` [`shared/robot_manager/config.go]. Then, the robot will need to be accepted by the user using one of the user servers by publishing an event to `HANDLE_REGISTERING_ROBOT_EVENT_FMT`. This is all assuming the robot's type is successfully defined in the program already.

Once the robot is registered, the fun part begins.

There are 3 parts to a robot, `the connection handler`, `the robot handler`, and `the robot`. All interfaces and structs below are defined in `shared/robot_types.go`.

#### The connection handler

The connection handler as described in `shared/robot_types.go` is the interface `RobotConnHandler`. 
- The base robot connection handler is `BaseRobotConnHandler`.

Essentially, the robot connection handler has methods `Start` and `Stop`. `Start` is called upon successful connection with the robomesh, and should be the method that contains the message queue processing. In other words, 

- `Start` is the go routine that will handle communications to and from the robot and the various services.

- `Stop` is just the cleanup method that is called whenever the go routine ends.

- `GetHandler` should return the robot handler for that robot.

- `GetDisconnectChannel` returns the channel that monitors whether the go routine is still running.

#### The robot handler

The robot handler is the main struct that deals with the logic of handling communication with various services. Once the robot is connected and the robot connection handler start routine is running, the robot handler is the primary way that the robot manager will communicate with the robot's go routine.

```go
type RobotHandler interface {
	GetRobot() Robot                                    // Access robot state for API responses and status checks
	SendMsg(msg Msg) error                              // Queue message for asynchronous processing by robot
	GetDeviceID() string                                // Get unique robot identifier for routing and logging
	GetIP() string                                      // Get current IP address for network diagnostics
	GetDisconnectChannel() chan bool                    // Get coordination channel for graceful shutdown
	QuickAction(w http.ResponseWriter, r *http.Request) // Perform immediate status check or health ping
	GET(w http.ResponseWriter, r *http.Request)         // Handle GET requests for robot state
	POST(w http.ResponseWriter, r *http.Request)        // Handle POST requests for robot actions
}
```

- `GetRobot` gets the robot's attributes. This should be a struct of the attributes of the robot but it shouldn't provide a way to modify the robot directly.
- `SendMsg` adds a message onto the message queue. The idea is that the robot's go routine will read messages off of the message channel and then act and respond appropriately.
  - `Msg` interface provides a way for services to communicate with the robot. There is a `DefaultMsg` struct for simpler implementation.
- `GetDeviceID` get's the id of the device.
- `GetIP` get's the ip of the device.
- `GetDisconnectChannel` get's the disconnect channel of the go routine. I don't know the importance of this.

The following methods are for integrating with the frontend webserver. The frontend webserver has 3 ways it can interact with the backend robot's go routine. 

- `QuickAction` is for any quick actions or info retrieval, such as a battery retrieval.
- `GET` is for when the user wants more information.
- `POST` is for when the user wants to give information to the go routine.

The `GET` and `POST` allow arguments to be passed in. The fetching process from the frontend should be programmed by you, but the `QuickAction` is just a GET request to some http path on the backend.