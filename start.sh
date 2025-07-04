#!/bin/bash

# 启动 Go 程序并使用 nohup 保持后台运行
nohup ./docker-registry-manager > output.log 2>&1 &

# 输出程序的 PID
echo "Go 程序已启动，PID: $!"
ps -p $! -o comm,%mem,rss
