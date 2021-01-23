# biliroaming-go-server ![workflows](https://github.com/JasonKhew96/biliroaming-go-server/workflows/Go/badge.svg)


## 需求
- 脑子
- Redis
- Nginx (可选)

## 使用方式
1. 安装并启用 Redis (https://redis.io/download)
2. 修改设置文件 `config.yaml`
3. 启用程序 (systemd/screen/nohup)

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

### screen
- 根据 linux 发行版执行安装 screen
- 程序路径执行 `screen -dmS "biliroaming-server" ./biliroaming-go-server`
- 连接 screen 程序 `screen -r biliroaming-server`，连上后 `ctrl+c` 关闭程序
- 断开 screen 程序连接，键盘按 `ctrl+a d`
- screen 列出所有 `screen -ls`

### nohup
- 程序路径执行 `nohup ./biliroaming-go-server &`
- 停止程序 `kill -9 1234`，1234 替换程序为 PID
