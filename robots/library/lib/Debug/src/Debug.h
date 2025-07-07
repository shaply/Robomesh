#ifndef ROBOMESH_DEBUG_H
#define ROBOMESH_DEBUG_H

#ifdef DEBUG

#include <Arduino.h>

#define DEBUG_PRINTLN(x) Serial.println(x)
#define DEBUG_PRINT(x) Serial.print(x)
#define DEBUG_PRINT_HEX(x) Serial.print(x, HEX)
#define DEBUG_PRINT_DEC(x) Serial.print(x, DEC)
#define DEBUG_PRINT_BIN(x) Serial.print(x, BIN)
#define DEBUG_PRINT_FLOAT(x) Serial.print(x, 6)
#define DEBUG_PRINT_CHAR(x) Serial.print((char)x)
#define DEBUG_PRINT_STRING(x) Serial.print(x)
#define DEBUG_PRINT_BUFFER(buf, len) \
    { \
        Serial.print("Buffer: "); \
        for (size_t i = 0; i < len; i++) { \
            Serial.print(buf[i], HEX); \
            if (i < len - 1) Serial.print(", "); \
        } \
        Serial.println(); \
    }
#define DEBUG_PRINT_ARRAY(arr, len) \
    { \
        Serial.print("Array: "); \
        for (size_t i = 0; i < len; i++) { \
            Serial.print(arr[i], HEX); \
            if (i < len - 1) Serial.print(", "); \
        } \
        Serial.println(); \
    }
#define DEBUG_PRINT_OBJECT(obj) \
    { \
        Serial.print("Object: "); \
        obj.printTo(Serial); \
        Serial.println(); \
    }
#define DEBUG_PRINT_ERROR(x) Serial.print("Error: "); Serial.println(x)
#define DEBUG_PRINT_WARNING(x) Serial.print("Warning: "); Serial.println(x)
#define DEBUG_PRINT_INFO(x) Serial.print("Info: "); Serial.println(x)

#else

#define DEBUG_PRINTLN(x)
#define DEBUG_PRINT(x)
#define DEBUG_PRINT_HEX(x)
#define DEBUG_PRINT_DEC(x)
#define DEBUG_PRINT_BIN(x)
#define DEBUG_PRINT_FLOAT(x)
#define DEBUG_PRINT_CHAR(x)
#define DEBUG_PRINT_STRING(x)
#define DEBUG_PRINT_BUFFER(buf, len)
#define DEBUG_PRINT_ARRAY(arr, len)
#define DEBUG_PRINT_OBJECT(obj)
#define DEBUG_PRINT_ERROR(x)
#define DEBUG_PRINT_WARNING(x)
#define DEBUG_PRINT_INFO(x)

#endif // DEBUG

#endif // ROBOMESH_DEBUG_H