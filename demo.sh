#!/bin/bash

# Docker Registry Manager 演示脚本

echo "=== Docker Registry Manager 演示 ==="
echo

# 检查是否已安装Go
if ! command -v go &> /dev/null; then
    echo "错误: 需要安装Go 1.21+才能运行此演示"
    echo "请访问 https://golang.org/dl/ 下载并安装Go"
    exit 1
fi

# 检查是否已安装Docker
if ! command -v docker &> /dev/null; then
    echo "警告: 未检测到Docker，某些演示功能可能无法使用"
fi

echo "1. 构建项目..."
make build

if [ $? -ne 0 ]; then
    echo "错误: 构建失败"
    exit 1
fi

echo "✓ 构建成功"
echo

echo "2. 设置环境..."
make setup

echo "✓ 环境设置完成"
echo

echo "3. 启动Docker Registry Manager..."
./build/docker-registry-manager -config config.yaml &
SERVER_PID=$!

# 等待服务启动
sleep 3

# 检查服务是否正在运行
if ! curl -s http://localhost:7000/v2/ > /dev/null; then
    echo "错误: 服务启动失败"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

echo "✓ 服务已启动在 http://localhost:7000"
echo

echo "4. 测试API端点..."

echo "  - 检查API版本支持:"
curl -s http://localhost:7000/v2/ | head -1
echo

echo "  - 获取仓库列表:"
curl -s http://localhost:7000/v2/_catalog | jq . 2>/dev/null || curl -s http://localhost:7000/v2/_catalog
echo

echo "  - 获取统计信息:"
curl -s http://localhost:7000/api/stats | jq . 2>/dev/null || curl -s http://localhost:7000/api/stats
echo

echo "✓ API测试完成"
echo

echo "=== 演示完成 ==="
echo
echo "服务正在运行，您可以："
echo "  • 访问 Web 界面: http://localhost:7000"
echo "  • 查看 API 文档: 参考 README.md"
echo "  • 测试 Docker 推送/拉取功能"
echo
echo "要停止服务，请运行: kill $SERVER_PID"
echo "或按 Ctrl+C 然后运行: pkill docker-registry-manager"
echo

# 如果安装了Docker，显示使用示例
if command -v docker &> /dev/null; then
    echo "Docker 使用示例:"
    echo "  # 推送镜像"
    echo "  docker tag hello-world localhost:7000/hello-world:latest"
    echo "  docker push localhost:7000/hello-world:latest"
    echo
    echo "  # 拉取镜像"
    echo "  docker pull localhost:7000/hello-world:latest"
    echo
fi

echo "服务进程ID: $SERVER_PID"

