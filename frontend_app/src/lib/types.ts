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

export interface RegisteredRobot {
  UUID: string;
  PublicKey: string;
  DeviceType: string;
  IsBlacklisted: boolean;
  CreatedAt: string;
}

export interface ActiveRobot {
  uuid: string;
  ip: string;
  device_type: string;
  session_jwt: string;
  pid: number;
  connected_at: number;
}

export interface PendingRobot {
  uuid: string;
  ip: string;
  device_type: string;
  public_key: string;
  requested_at: number;
}
