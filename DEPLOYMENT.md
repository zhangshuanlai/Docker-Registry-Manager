# Docker Registry Manager 部署指南

本文档提供了Docker Registry Manager的详细部署指南，包括开发环境、生产环境和容器化部署。

## 目录

- [系统要求](#系统要求)
- [快速开始](#快速开始)
- [开发环境部署](#开发环境部署)
- [生产环境部署](#生产环境部署)
- [容器化部署](#容器化部署)
- [反向代理配置](#反向代理配置)
- [安全配置](#安全配置)
- [监控和日志](#监控和日志)
- [故障排除](#故障排除)

## 系统要求

### 最低要求
- **操作系统**: Linux, macOS, Windows
- **Go版本**: 1.21+
- **内存**: 512MB RAM
- **存储**: 1GB 可用空间
- **网络**: 开放端口 5000（可配置）

### 推荐配置
- **内存**: 2GB+ RAM
- **存储**: 10GB+ 可用空间（取决于镜像数量）
- **CPU**: 2+ 核心

## 快速开始

### 1. 下载和构建

```bash
# 克隆项目
git clone <repository-url>
cd docker-registry-manager

# 构建项目
make build

# 运行演示
./demo.sh
```

### 2. 访问服务

- **Web界面**: http://localhost:5000
- **API端点**: http://localhost:5000/v2/

## 开发环境部署

### 1. 环境准备

```bash
# 安装Go (如果未安装)
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# 验证安装
go version
```

### 2. 项目设置

```bash
# 克隆项目
git clone <repository-url>
cd docker-registry-manager

# 安装依赖
make install-deps

# 设置开发环境
make setup

# 构建项目
make build
```

### 3. 配置文件

编辑 `config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 5000
  read_timeout: 30s
  write_timeout: 30s

storage:
  type: "filesystem"
  path: "./data"

logging:
  level: "debug"  # 开发环境使用debug级别
  format: "text"  # 开发环境使用文本格式

web:
  enabled: true
  title: "Docker Registry Manager (Dev)"
```

### 4. 启动服务

```bash
# 开发模式启动
make dev

# 或手动启动
./build/docker-registry-manager -config config.yaml
```

## 生产环境部署

### 1. 系统准备

```bash
# 创建专用用户
sudo useradd -r -s /bin/false docker-registry

# 创建目录
sudo mkdir -p /opt/docker-registry-manager
sudo mkdir -p /var/lib/docker-registry
sudo mkdir -p /var/log/docker-registry

# 设置权限
sudo chown docker-registry:docker-registry /var/lib/docker-registry
sudo chown docker-registry:docker-registry /var/log/docker-registry
```

### 2. 部署应用

```bash
# 构建发布版本
make release

# 复制文件到生产目录
sudo cp build/docker-registry-manager /opt/docker-registry-manager/
sudo cp -r web /opt/docker-registry-manager/
sudo cp config.yaml /opt/docker-registry-manager/production.yaml

# 设置权限
sudo chown -R docker-registry:docker-registry /opt/docker-registry-manager
sudo chmod +x /opt/docker-registry-manager/docker-registry-manager
```

### 3. 生产配置

编辑 `/opt/docker-registry-manager/production.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 5000
  read_timeout: 60s
  write_timeout: 60s

storage:
  type: "filesystem"
  path: "/var/lib/docker-registry"

registry:
  realm: "Production Docker Registry"
  service: "docker-registry-manager"

logging:
  level: "warn"
  format: "json"

web:
  enabled: true
  title: "Docker Registry Manager"

cors:
  enabled: true
  allowed_origins: ["https://your-domain.com"]
  allowed_methods: ["GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"]
  allowed_headers: ["*"]
```

### 4. 系统服务配置

创建 systemd 服务文件 `/etc/systemd/system/docker-registry-manager.service`:

```ini
[Unit]
Description=Docker Registry Manager
After=network.target

[Service]
Type=simple
User=docker-registry
Group=docker-registry
WorkingDirectory=/opt/docker-registry-manager
ExecStart=/opt/docker-registry-manager/docker-registry-manager -config production.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# 安全设置
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/docker-registry

[Install]
WantedBy=multi-user.target
```

### 5. 启动服务

```bash
# 重新加载systemd配置
sudo systemctl daemon-reload

# 启用并启动服务
sudo systemctl enable docker-registry-manager
sudo systemctl start docker-registry-manager

# 检查状态
sudo systemctl status docker-registry-manager
```

## 容器化部署

### 1. Dockerfile

创建 `Dockerfile`:

```dockerfile
# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

# 安装依赖并构建
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o docker-registry-manager ./cmd

# 运行阶段
FROM alpine:latest

# 安装CA证书
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# 复制构建产物
COPY --from=builder /app/docker-registry-manager .
COPY --from=builder /app/web ./web
COPY --from=builder /app/config.yaml .

# 创建数据目录
RUN mkdir -p data/{blobs,repositories,uploads}

# 暴露端口
EXPOSE 5000

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:5000/v2/ || exit 1

# 启动命令
CMD ["./docker-registry-manager"]
```

### 2. Docker Compose

创建 `docker-compose.yml`:

```yaml
version: '3.8'

services:
  docker-registry-manager:
    build: .
    ports:
      - "5000:5000"
    volumes:
      - registry_data:/root/data
      - ./config.yaml:/root/config.yaml:ro
    environment:
      - REGISTRY_LOG_LEVEL=info
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:5000/v2/"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 40s

volumes:
  registry_data:
    driver: local
```

### 3. 构建和运行

```bash
# 构建镜像
docker build -t docker-registry-manager .

# 使用Docker Compose运行
docker-compose up -d

# 检查状态
docker-compose ps
docker-compose logs -f
```

## 反向代理配置

### Nginx 配置

创建 `/etc/nginx/sites-available/docker-registry`:

```nginx
upstream docker-registry-manager {
    server 127.0.0.1:5000;
}

server {
    listen 80;
    server_name registry.your-domain.com;
    
    # 重定向到HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name registry.your-domain.com;

    # SSL配置
    ssl_certificate /path/to/ssl/cert.pem;
    ssl_certificate_key /path/to/ssl/key.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;

    # 客户端最大上传大小
    client_max_body_size 1G;

    # 代理配置
    location / {
        proxy_pass http://docker-registry-manager;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # 超时设置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # 特殊处理Docker Registry API
    location /v2/ {
        proxy_pass http://docker-registry-manager;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # 禁用缓冲以支持流式上传
        proxy_buffering off;
        proxy_request_buffering off;
    }
}
```

### Apache 配置

```apache
<VirtualHost *:80>
    ServerName registry.your-domain.com
    Redirect permanent / https://registry.your-domain.com/
</VirtualHost>

<VirtualHost *:443>
    ServerName registry.your-domain.com
    
    # SSL配置
    SSLEngine on
    SSLCertificateFile /path/to/ssl/cert.pem
    SSLCertificateKeyFile /path/to/ssl/key.pem
    
    # 代理配置
    ProxyPreserveHost On
    ProxyPass / http://127.0.0.1:5000/
    ProxyPassReverse / http://127.0.0.1:5000/
    
    # 设置头部
    ProxyPassReverse / http://127.0.0.1:5000/
    ProxyPassReverseMatch ^(.*) http://127.0.0.1:5000$1
</VirtualHost>
```

## 安全配置

### 1. 防火墙设置

```bash
# UFW (Ubuntu)
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable

# iptables
sudo iptables -A INPUT -p tcp --dport 22 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 5000 -j DROP  # 阻止直接访问
```

### 2. SSL/TLS 配置

使用 Let's Encrypt 获取免费SSL证书:

```bash
# 安装certbot
sudo apt-get install certbot python3-certbot-nginx

# 获取证书
sudo certbot --nginx -d registry.your-domain.com

# 自动续期
sudo crontab -e
# 添加: 0 12 * * * /usr/bin/certbot renew --quiet
```

### 3. 访问控制

在配置文件中添加基本认证:

```yaml
auth:
  enabled: true
  type: "basic"
  realm: "Docker Registry"
  users:
    - username: "admin"
      password: "$2a$10$..." # bcrypt哈希
```

## 监控和日志

### 1. 日志配置

```yaml
logging:
  level: "info"
  format: "json"
  output: "/var/log/docker-registry/app.log"
  max_size: 100  # MB
  max_backups: 5
  max_age: 30    # days
```

### 2. 监控指标

添加Prometheus指标端点:

```yaml
metrics:
  enabled: true
  path: "/metrics"
  port: 9090
```

### 3. 健康检查

```bash
# 简单健康检查
curl -f http://localhost:5000/v2/ || exit 1

# 详细健康检查脚本
#!/bin/bash
HEALTH_URL="http://localhost:5000/v2/"
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" $HEALTH_URL)

if [ $RESPONSE -eq 200 ]; then
    echo "Service is healthy"
    exit 0
else
    echo "Service is unhealthy (HTTP $RESPONSE)"
    exit 1
fi
```

## 故障排除

### 常见问题

1. **服务无法启动**
   ```bash
   # 检查日志
   sudo journalctl -u docker-registry-manager -f
   
   # 检查配置
   ./docker-registry-manager -config config.yaml -validate
   ```

2. **权限问题**
   ```bash
   # 检查文件权限
   ls -la /var/lib/docker-registry
   
   # 修复权限
   sudo chown -R docker-registry:docker-registry /var/lib/docker-registry
   ```

3. **端口冲突**
   ```bash
   # 检查端口占用
   sudo netstat -tlnp | grep :5000
   
   # 修改配置文件中的端口
   ```

4. **存储空间不足**
   ```bash
   # 检查磁盘空间
   df -h /var/lib/docker-registry
   
   # 清理旧数据
   find /var/lib/docker-registry -type f -mtime +30 -delete
   ```

### 调试模式

启用调试模式:

```yaml
logging:
  level: "debug"
  format: "text"
```

### 性能调优

1. **增加文件描述符限制**
   ```bash
   # 临时设置
   ulimit -n 65536
   
   # 永久设置 /etc/security/limits.conf
   docker-registry soft nofile 65536
   docker-registry hard nofile 65536
   ```

2. **调整Go运行时参数**
   ```bash
   export GOMAXPROCS=4
   export GOGC=100
   ```

3. **优化存储配置**
   ```yaml
   storage:
     type: "filesystem"
     path: "/var/lib/docker-registry"
     options:
       cache_size: "1GB"
       max_concurrent_uploads: 10
   ```

## 备份和恢复

### 备份脚本

```bash
#!/bin/bash
BACKUP_DIR="/backup/docker-registry"
DATA_DIR="/var/lib/docker-registry"
DATE=$(date +%Y%m%d_%H%M%S)

# 创建备份目录
mkdir -p $BACKUP_DIR

# 停止服务
sudo systemctl stop docker-registry-manager

# 创建备份
tar -czf $BACKUP_DIR/registry_backup_$DATE.tar.gz -C $DATA_DIR .

# 启动服务
sudo systemctl start docker-registry-manager

echo "Backup completed: $BACKUP_DIR/registry_backup_$DATE.tar.gz"
```

### 恢复脚本

```bash
#!/bin/bash
BACKUP_FILE=$1
DATA_DIR="/var/lib/docker-registry"

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file>"
    exit 1
fi

# 停止服务
sudo systemctl stop docker-registry-manager

# 备份当前数据
mv $DATA_DIR $DATA_DIR.old

# 创建新目录
mkdir -p $DATA_DIR

# 恢复数据
tar -xzf $BACKUP_FILE -C $DATA_DIR

# 设置权限
sudo chown -R docker-registry:docker-registry $DATA_DIR

# 启动服务
sudo systemctl start docker-registry-manager

echo "Restore completed from: $BACKUP_FILE"
```

---

如需更多帮助，请参考项目README.md或提交Issue。

