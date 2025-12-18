#cmake -DCMAKE_VERBOSE_MAKEFILE=ON .. 查看详细信息
CMAKE_MINIMUM_REQUIRED(VERSION 3.12)
PROJECT(ha)

# 寻找 Go 编译器
find_program(GO_EXECUTABLE go REQUIRED)
if(NOT GO_EXECUTABLE)
    message(FATAL_ERROR "Go compiler not found")
else()	
	get_directory_property(MyDir SOURCE_DIR)
endif()

# 添加 ExternalProject 模块
include(ExternalProject)

set(GO_LIBRARY ${CMAKE_BINARY_DIR}/lib)

ExternalProject_Add(ha
    PREFIX "${MyDir}/build"
    SOURCE_DIR "${MyDir}"  # 使用定义的变量
    CONFIGURE_COMMAND ""
    BUILD_COMMAND go build -buildmode=c-shared -o ${CMAKE_LIBRARY_OUTPUT_DIRECTORY}/libmyha.so ha.go
    INSTALL_COMMAND ""
    BUILD_IN_SOURCE 1
	BUILD_ALWAYS 1 
)

add_custom_target(clean_go_build
    COMMAND ${CMAKE_COMMAND} -E remove -f ${CMAKE_LIBRARY_OUTPUT_DIRECTORY}/libha.h
)