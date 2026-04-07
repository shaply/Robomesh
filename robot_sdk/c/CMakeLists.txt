cmake_minimum_required(VERSION 3.16)
project(robomesh_sdk C)

set(CMAKE_C_STANDARD 11)

find_package(OpenSSL REQUIRED)
find_package(Threads REQUIRED)

# Library
add_library(robomesh STATIC src/robomesh.c)
target_include_directories(robomesh PUBLIC include)
target_link_libraries(robomesh PUBLIC OpenSSL::Crypto Threads::Threads)

# Test robot example
add_executable(test_robot examples/test_robot.c)
target_link_libraries(test_robot robomesh)

# Integration test
add_executable(test_integration tests/test_integration.c)
target_link_libraries(test_integration robomesh)
