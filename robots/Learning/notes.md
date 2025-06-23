
# Arduino Uno R4 WiFi - PlatformIO Setup Notes

## Hardware Info
- **Board**: Arduino Uno R4 WiFi
- **Port**: `/dev/cu.usbmodemF0F5BD51D1702`
- **Platform**: Renesas RA (ARM Cortex-M4)
- **RAM**: 32KB
- **Flash**: 256KB

## Initial Setup

### 1. Install PlatformIO
```bash
pip install platformio
```

### 2. Initialize Project for Arduino Uno R4 WiFi
```bash
pio project init --board uno_r4_wifi
```

### 3. Install Required Platform
```bash
pio platform install renesas-ra
```

### 4. Configure VS Code Integration
```bash
pio project init --ide vscode
```

## Project Structure
```
Project1/
├── platformio.ini       # Main configuration file
├── src/                 # Source code (.ino files go here)
├── lib/                 # Project libraries
├── include/             # Header files
├── test/                # Unit tests
└── .vscode/             # VS Code configuration
    ├── c_cpp_properties.json  # IntelliSense config
    ├── settings.json           # VS Code settings
    ├── extensions.json         # Recommended extensions
    └── launch.json             # Debug configuration
```

## Configuration Files

### platformio.ini
```ini
[env:uno_r4_wifi]
platform = renesas-ra
board = uno_r4_wifi
framework = arduino
upload_port = /dev/cu.usbmodemF0F5BD51D1702
monitor_port = /dev/cu.usbmodemF0F5BD51D1702
monitor_speed = 9600
```
- Figured out using `arduino-cli board list`

## Essential Commands

### Building and Uploading
- `pio run` - Build the project
- `pio run --target upload` - Build and upload to board
- `pio run --target clean` - Clean build files

### Monitoring and Debugging
- `pio device monitor` - Start serial monitor
- `pio device list` - List available devices
- `pio run --target compiledb` - Generate compilation database for IntelliSense

### Project Management
- `pio project init --board <board_name>` - Initialize new project
- `pio project init --ide vscode` - Generate VS Code configuration
- `pio lib search <keyword>` - Search for libraries
- `pio lib install <library>` - Install library

## VS Code Setup Issues & Solutions

### If you get `#include` errors:
1. Make sure PlatformIO IDE extension is installed
2. Run: `pio project init --ide vscode`
3. Reload VS Code window
4. Check that `.vscode/c_cpp_properties.json` exists

### Key VS Code Extensions:
- PlatformIO IDE (`platformio.platformio-ide`)
- C/C++ (`ms-vscode.cpptools`)

## Code Examples

### Basic Blink (1 second on, 5 seconds off)
```cpp
#include <Arduino.h>

void setup() {
  Serial.begin(9600);
  pinMode(13, OUTPUT);
}

void loop() {
  digitalWrite(13, HIGH);   // Turn LED on
  delay(1000);              // Wait 1 second
  digitalWrite(13, LOW);    // Turn LED off
  delay(5000);              // Wait 5 seconds
}
```

## Troubleshooting

### Common Issues:
1. **Port not found**: Check device connection and port in `platformio.ini`
2. **Include errors**: Run `pio project init --ide vscode` and reload VS Code
3. **Build fails**: Make sure Renesas RA platform is installed
4. **Upload fails**: Try pressing reset button on board during upload

### Useful Debugging:
- Add `-v` flag for verbose output: `pio run -v`
- Check serial output: `pio device monitor`
- Verify board connection: `pio device list`

## Notes for Next Time:
- Always use `pio project init --board uno_r4_wifi` for R4 WiFi projects
- Remember to set the correct port in `platformio.ini`
- Use `pio project init --ide vscode` to fix IntelliSense issues
- The R4 WiFi uses Renesas RA platform, not AVR like classic Uno
- `arduino-cli board list` shows the connected boards

## Notes on programming
- Include `Arduino.h`
- Always have `void setup` and `void loop`