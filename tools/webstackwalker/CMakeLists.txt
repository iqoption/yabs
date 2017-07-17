cmake_minimum_required(VERSION 3.7)
project(webstackwalker)

set(CMAKE_CXX_STANDARD 14)


include_directories(3rdparty/rapidjson/include)
set(SOURCE_FILES main.cpp)
add_executable(webstackwalker ${SOURCE_FILES})