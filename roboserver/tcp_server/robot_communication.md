## Robot TCP Communication Format

`REGISTER`: Registers the robot to the server
- Input:
  - `REGISTER <RobotType> <RobotID>`
    - `<RobotType>`: String, max 32 chars, name
    - `<RobotID>`: String, max 32 chars, self generated ID, maybe make it persist later on
- Response: `OK REGISTER` on success, `ERROR REGISTER` on failure.

`UNREGISTER`: Unregisters the robot from the server
- Input:
  - `UNREGISTER [<RobotId]`
    - `<RobotId>` is used when specifying a robot of another device, the id is the same as the one in the server.