export interface BaseRobot {
  device_id: string;
  name: string;
  ip: string;
  robot_type: string;
  status: string;
  last_seen: number;
}

export interface RegisteringRobotEvent {
  device_id: string;
  ip?: string;
  robot_type: string;
}