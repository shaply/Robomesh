cmake_minimum_required(VERSION 3.16)
project(robomesh_sdk C)

set(CMAKE_C_STANDARD 11)

find_package(OpenSSL REQUIRED)
find_package(Threads REQUIRED)

# ── Core TCP library ─────────────────────────────────────────
add_library(robomesh STATIC src/robomesh.c)
target_include_directories(robomesh PUBLIC include)
target_link_libraries(robomesh PUBLIC OpenSSL::Crypto Threads::Threads)

# ── UDP library ──────────────────────────────────────────────
add_library(robomesh_udp STATIC src/robomesh_udp.c)
target_include_directories(robomesh_udp PUBLIC include)
target_link_libraries(robomesh_udp PUBLIC OpenSSL::Crypto Threads::Threads)

# ── MQTT library (optional, requires libmosquitto) ───────────
find_library(MOSQUITTO_LIB mosquitto)
find_path(MOSQUITTO_INCLUDE mosquitto.h)
if(MOSQUITTO_LIB AND MOSQUITTO_INCLUDE)
    add_library(robomesh_mqtt STATIC src/robomesh_mqtt.c)
    target_include_directories(robomesh_mqtt PUBLIC include ${MOSQUITTO_INCLUDE})
    target_link_libraries(robomesh_mqtt PUBLIC OpenSSL::Crypto Threads::Threads ${MOSQUITTO_LIB})

    # MQTT test robot example
    add_executable(test_robot_mqtt examples/test_robot_mqtt.c)
    target_link_libraries(test_robot_mqtt robomesh_mqtt robomesh)

    message(STATUS "libmosquitto found — MQTT client will be built")
else()
    message(STATUS "libmosquitto not found — MQTT client will NOT be built")
endif()

# ── TCP test robot example ───────────────────────────────────
add_executable(test_robot examples/test_robot.c)
target_link_libraries(test_robot robomesh)

# ── UDP test robot example ───────────────────────────────────
add_executable(test_robot_udp examples/test_robot_udp.c)
target_link_libraries(test_robot_udp robomesh_udp robomesh)

# ── Integration tests ────────────────────────────────────────
add_executable(test_integration tests/test_integration.c)
target_link_libraries(test_integration robomesh)
