#!/bin/bash
# Handler entry point for robot type: test_robot
#
# A network diagnostics robot that can run wifi speed tests, ping,
# and report signal strength metrics.

exec python3 "$(dirname "$0")/handler.py"
