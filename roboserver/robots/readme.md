To Create a robot, you must complete 3 interfaces:
- `RobotConnHandler`
  - `Start`: Leads to an indefinite while loop, otherwise, the robot will be disconnected.
- `RobotHandler`
- `Robot`: Must embed `BaseRobot`

`BaseRobot` has default implementations for all of these things. 