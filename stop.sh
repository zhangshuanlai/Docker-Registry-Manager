#!/bin/bash

# 获取 Go 程序的 PID
PID=$(ps -ef | grep docker-registry-manager | grep -v grep | awk '{print $2}')

if [ -z "$PID" ]; then
    echo "Go 程序未运行."
else
    # 杀死 Go 程序
    kill -9 $PID
    echo "Go 程序已停止，PID: $PID"
fi
