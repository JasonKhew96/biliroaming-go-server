# biliroaming-go-server [![go report](https://goreportcard.com/badge/github.com/JasonKhew96/biliroaming-go-server)](https://goreportcard.com/report/github.com/JasonKhew96/biliroaming-go-server) [![workflows](https://github.com/JasonKhew96/biliroaming-go-server/workflows/Go/badge.svg)](https://github.com/JasonKhew96/biliroaming-go-server/actions)


## 需求
- 脑子
- PostgreSQL
- Nginx (可选)

## 使用方式
1. 安装并启用 [PostgreSQL](https://www.postgresql.org/)
2. 修改设置文件 `config.yaml`
3. 修改 Nginx 设置文件
4. 启用程序 (systemd/screen/nohup)

### systemd
- 创建文件 `/etc/systemd/system/biliroaming-go-server.service` (可以自选名字)
- `ExecStart` 请替换为正确的路径
```
[Unit]
Description=哔哩漫游代理

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
    # 域名设置
    server_name bili.example.com;

    # http
    # listen 80;
    # listen [::]:80;

    # https
    listen 443 ssl http2;
    listen [::]:443 ssl http2;

    # 限制客户端请求大小
    client_max_body_size 8M;

    location / {
        # 转发
        proxy_pass http://127.0.0.1:23333
        proxy_set_header Host $proxy_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

        if ($http_x_from_biliroaming ~ "^$") {  # 模块请求都会带上X-From-Biliroaming的请求头，为了防止被盗用，可以加上请求头判断
            return 403;
        }

        # cloudflare 转发真实 IP
        # https://support.cloudflare.com/hc/en-us/articles/200170786-Restoring-original-visitor-IPs-logging-visitor-IP-addresses
        set_real_ip_from 103.21.244.0/22;
        set_real_ip_from 103.22.200.0/22;
        set_real_ip_from 103.31.4.0/22;
        set_real_ip_from 104.16.0.0/12;
        set_real_ip_from 108.162.192.0/18;
        set_real_ip_from 131.0.72.0/22;
        set_real_ip_from 141.101.64.0/18;
        set_real_ip_from 162.158.0.0/15;
        set_real_ip_from 172.64.0.0/13;
        set_real_ip_from 173.245.48.0/20;
        set_real_ip_from 188.114.96.0/20;
        set_real_ip_from 190.93.240.0/20;
        set_real_ip_from 197.234.240.0/22;
        set_real_ip_from 198.41.128.0/17;
        set_real_ip_from 2400:cb00::/32;
        set_real_ip_from 2606:4700::/32;
        set_real_ip_from 2803:f800::/32;
        set_real_ip_from 2405:b500::/32;
        set_real_ip_from 2405:8100::/32;
        set_real_ip_from 2c0f:f248::/32;
        set_real_ip_from 2a06:98c0::/29;

        # 二选一
        real_ip_header CF-Connecting-IP;
        # real_ip_header X-Forwarded-For;
    }

    # RSA certificate
    ssl_certificate     /etc/nginx/ssl/example.com/full.pem;  # 证书路径
    ssl_certificate_key /etc/nginx/ssl/example.com/key.pem;
    ssl_protocols       TLSv1 TLSv1.1 TLSv1.2;
    ssl_ciphers         HIGH:!aNULL:!MD5;
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

## TODO
- 支持 Docker