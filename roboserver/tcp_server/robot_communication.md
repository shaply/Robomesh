## TCP Server

The TCP server accepts connections from robots and will forward any message it receives to the Go handler routine's message channel from the Robot Manager. The TCP server forwards messages line by line, and each line, the TCP server will wait till the Go routine reads the message by taking it off the message channel.

The TCP message has a `GetConn()` function to get the connection and the Go routine is responsible for writing back a response to the robot. The `Source` will always be `TCP_SERVER` and the `Msg` will be the line that was read.

There is handling for messages from IPs not registered with robot manager, please just look at code for this.

However, if you want to change the way the messages from the robot are parsed, please look at the `TRANSFER` command.

## Robot TCP Communication Format

`REGISTER`: Registers the robot to the server. Will add the robot to the robot manager as well as call the robot type's connection handler's `Start` and `Stop` methods.
- Input:
  - `REGISTER <RobotType> <RobotID>`
    - `<RobotType>`: String, max 32 chars, name
    - `<RobotID>`: String, max 32 chars, self generated ID, maybe make it persist later on
- Response: `OK` on success, `ERROR REGISTER` on failure.
  - `ERROR NO_ROBOTYPE_CONN_HANDLER`: The robot doesn't have a connection handler.
  - `ERROR CREATE_CONN_HANDLER`: Creating the connection handler was a failure.
  - `ERROR ROBOT_ALREADY_EXISTS`: The robot is already in the system.
  - `ERROR NO_DISCONNECT_CHANNEL`: The robot handler doesn't have a disconnect channel.
  - `ERROR UNKNOWN`: The error is ambiguous.

`TRANSFER`: Stalls the TCP connection handler (so the TCP server stops reading), and gives the reading and writing functionality solely to the robot handling go routine. The message it sends to the go routine will have a reply channel and a message of TRANSFER and when the Go routine wants to stop parsing the TCP connection, it just needs to write to the reply channel.