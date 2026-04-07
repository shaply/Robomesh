#!/bin/bash
# Handler entry point for robot type: [ROBOT_TYPE]
#
# Environment variables available:
#   ROBOT_UUID        - The robot's unique identifier
#   ROBOT_DEVICE_TYPE - The device type string
#   ROBOT_IP          - The robot's IP address
#   ROBOT_SESSION_ID  - The current session identifier
#
# Communication protocol (stdin/stdout JSON):
#
# INCOMING (stdin - from roboserver to handler):
#   {"type":"connect","uuid":"...","device_type":"...","ip":"...","session_id":"..."}
#   {"type":"incoming","uuid":"...","payload":"..."}
#   {"type":"disconnect","uuid":"...","reason":"..."}
#   {"type":"event","event_type":"...","data":{...}}
#   {"type":"heartbeat","event_type":"...","data":{...}}
#
# OUTGOING (stdout - from handler to roboserver):
#   {"target":"robot","id":"1","data":{...}}           - Send data to robot
#   {"target":"database","id":"2","method":"get_robot","data":"uuid"} - Query database
#   {"target":"event_bus","id":"3","method":"event.type","data":{...}} - Publish event
#   {"target":"config","id":"4","method":"forward_heartbeats","data":true} - Configure
#   {"target":"config","id":"5","method":"subscribe","data":"event.type"} - Subscribe to events
#   {"target":"connect_robot","id":"6","data":{"port":8888,"protocol":"tcp"}} - Reverse connect
#
# This script is a thin wrapper. Replace the python3 call below with your
# language of choice (node, python, go binary, etc.).

exec python3 "$(dirname "$0")/handler.py"
