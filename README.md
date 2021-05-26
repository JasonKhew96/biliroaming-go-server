# biliroaming-go-server [![go report](https://goreportcard.com/badge/github.com/JasonKhew96/biliroaming-go-server)](https://goreportcard.com/report/github.com/JasonKhew96/biliroaming-go-server) [![workflows](https://github.com/JasonKhew96/biliroaming-go-server/workflows/Go/badge.svg)](https://github.com/JasonKhew96/biliroaming-go-server/actions)


## 需求
- 脑子
- PostgreSQL
- Nginx (可选)
- Docker (可选)

## Docker
见 [JasonKhew96/biliroaming-go-server-docker](https://github.com/JasonKhew96/biliroaming-go-server-docker)

## 使用方式
1. 安装并启用 [PostgreSQL](https://www.postgresql.org/)
2. 设置 环境变量，见 [production.env.example](production.env.example)
3. 修改 Nginx 设置文件
4. 启用程序 (systemd/screen/nohup)

### systemd
- 创建文件 `/etc/systemd/system/biliroaming-go-server.service` (可以自选名字)
- `ExecStart` 请替换为正确的路径
```
[Unit]
Description=哔哩漫游代理服务

[Service]
ExecStart=/root/server/biliroaming-go-server

[Install]
WantedBy=multi-user.target
```
- 刷新后台程序 `systemctl daemon-reload`
- 启用后台程序 `systemctl enable biliroaming-go-server.service`
- 禁用后台程序 `systemctl disable biliroaming-go-server.service`
- 启动后台程序 `systemctl start biliroaming-go-server.service`
- 停止后台程序 `systemctl stop biliroaming-go-server.service`
- 检查后台程序状态 `systemctl status biliroaming-go-server.service`

### Nginx 端口转发
- 创建文件 `/etc/nginx/sites-available/bili.example.com` (文件名可选，域名比较方便)
```
server {
    # https
    listen  443 ssl;
    listen  [::]:443 ssl;

    server_name         bili.example.com;
    
    # 证书
    ssl_certificate     /etc/nginx/certs/site.crt;
    ssl_certificate_key /etc/nginx/certs/site.key;

    # 限制客户端请求大小
    client_max_body_size 1M;

    # 获取 CLOUDFLARE 真实 IP
    # https://support.cloudflare.com/hc/en-us/articles/200170786-Restoring-original-visitor-IPs
    # https://www.cloudflare.com/ips/
    # ipv4
    set_real_ip_from    173.245.48.0/20;
    set_real_ip_from    103.21.244.0/22;
    set_real_ip_from    103.22.200.0/22;
    set_real_ip_from    103.31.4.0/22;
    set_real_ip_from    141.101.64.0/18;
    set_real_ip_from    108.162.192.0/18;
    set_real_ip_from    190.93.240.0/20;
    set_real_ip_from    188.114.96.0/20;
    set_real_ip_from    197.234.240.0/22;
    set_real_ip_from    198.41.128.0/17;
    set_real_ip_from    162.158.0.0/15;
    set_real_ip_from    172.64.0.0/13;
    set_real_ip_from    131.0.72.0/22;
    set_real_ip_from    104.16.0.0/13;
    set_real_ip_from    104.24.0.0/14;
    # ipv6
    set_real_ip_from    2400:cb00::/32;
    set_real_ip_from    2606:4700::/32;
    set_real_ip_from    2803:f800::/32;
    set_real_ip_from    2405:b500::/32;
    set_real_ip_from    2405:8100::/32;
    set_real_ip_from    2a06:98c0::/29;
    set_real_ip_from    2c0f:f248::/32;

    # 2 选 1
    real_ip_header CF-Connecting-IP;
    # real_ip_header X-Forwarded-For;

    location / {
        proxy_pass  http://127.0.0.1:80;
    }
}
```

### screen
- 根据 linux 发行版执行安装 screen
- 程序路径执行 `screen -dmS "biliroaming-server" ./biliroaming-go-server`
- 连接 screen 程序 `screen -r biliroaming-server`，连上后 `ctrl+c` 关闭程序
- 断开 screen 程序连接，键盘按 `ctrl+a d`
- screen 列出所有 `screen -ls`

### nohup
- 程序路径执行 `nohup ./biliroaming-go-server &`
- 停止程序 `kill -9 1234`，1234 替换程序为 PID
