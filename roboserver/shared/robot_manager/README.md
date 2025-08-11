## Registration Flow

1. Publishes

    ```
    {
        event type: REGISTERING_ROBOT_EVENT "robot_manager.registering_robot"
        event data: RegisteringRobot {
            device_id: ...
            ip: ...
            robot_type: ...
        }
    }
    ```

2. To accept or deny a registering robot. Publish 

    '''
    {
        event type: HANDLE_REGISTERING_ROBOT_EVENT_FMT f"register.{device id}:{ip}:{robot type}"
        event data: "yes" or "no"
    }
    '''

    However, you can turn the event data into `RegisteringRobot` and then call `RegisteringRobot.HandleRegister(event_bus, acceptance bool)` as defined in `registration.go`.