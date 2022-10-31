<h1 align="center">项目简介</h1>

<p align="center">
本项目使用gin、gorm和ssh、sftp开发。旨在编写一个轻量，易用，多平台的运维项目。
现阶段目的是做一个阉割版的xshell并简单的实现ansible或者saltstack的部分功能。
</p>

<p align="center">
  <a href="https://github.com/ssbeatty/oms/blob/dev/LICENSE">
    <img src="https://img.shields.io/github/license/ssbeatty/oms" alt="license">
  </a>
  <a href="https://github.com/ssbeatty/oms/releases">
    <img src="https://img.shields.io/github/v/release/ssbeatty/oms?color=blueviolet&include_prereleases" alt="release">
  </a>
  <a href="https://github.com/ssbeatty/oms/actions">
    <img src="https://github.com/ssbeatty/oms/workflows/BUILD Linux/badge.svg" alt="action">
  </a>
  <a href="https://goreportcard.com/report/github.com/ssbeatty/oms">
  <img src="https://goreportcard.com/badge/github.com/ssbeatty/oms" alt="GoReportCard">
  </a>
</p>

<p align="center">
  <a href="https://github.com/lixin59/omsUI">UI</a>
  ·
  <a href="https://wang918562230.gitbook.io/ssbeattyoms-wen-dang/">文档</a>
  ·
  <a href="https://github.com/ssbeatty/oms/releases">下载</a>
  ·
  <a href="https://wang918562230.gitbook.io/ssbeattyoms-wen-dang/">开始使用</a>
</p>

### API文档

运行后访问 http://127.0.0.1:9090/swagger/index.html

[swagger](./docs/swagger.json)

### 使用说明
1. 安装编译
```shell script
# clone
git clone --recurse-submodules https://github.com/ssbeatty/oms.git

# build frontend
cd web/omsUI
yarn && yarn build

# 打包 oms
# linux
go build -o oms cmd/omsd/main.go
# win
go build -o oms.exe cmd/omsd/main.go
```

2. 启动 创建config.yaml在可执行文件同级 运行即可
```shell script
# configs/config.yaml.example
# 支持mysql postgres sqlite(默认, 仅调试)
app:
  name: oms
  addr: 127.0.0.1
  port: 9090
  mode: dev
  run_start: false # 是否在运行时打开浏览器 windows
  temp_date: 336h  # 执行日志的保存时间 默认14天

db:
  driver: postgres
  user: root
  password: 123456
  dsn: 127.0.0.1:3306
  db_name: oms
```

3. 注册为服务
```shell script
# 支持windows/linux/macos

oms --action install --config config.yaml

# 取消注册
oms --action uninstall 
```

> 注意注册为服务程序的运行目录会改变比如windows为C:/System32, 因此要修改配置中data_path为绝对路径。
> logger为相对路径时放在data_path下, 为绝对路径时在指定的路径。

### 目前已经实现的功能
1. web界面[omsUI](https://github.com/lixin59/omsUI/blob/master/README.md)
2. 隧道, 类似`ssh`的`-L`和`-R`
3. cron任务和exec任务的管理
4. ssh命令批量执行
5. 文件批量的上传 流式传输支持大文件
6. 基于`sftp`文件浏览器
7. 基于novnc的vnc viewer
8. 类似playbook的编排任务


### 环境变量
```shell
ENV_SSH_DIAL_TIMEOUT = 30  # ssh连接超时时间 单位秒
ENV_SSH_RW_TIMEOUT = 20    # ssh读写超时时间 单位秒
ENV_SSH_CMD_TIMEOUT = 120  # 执行命令时命令最长的超时时间 单位秒
```


### deploy
> docker-compose.yaml
```yaml
version: '2.3'

services:
  oms:
    image: ghcr.io/ssbeatty/oms/oms:v0.6.8
    restart: always
    extra_hosts:
      - "host.docker.internal:host-gateway"
    ports:
      - "9090:9090"
    volumes:
      - ./data:/opt/oms/data
      - ./config:/etc/oms
```

### 如何开发
1. 修改依赖之后执行
```shell
go mod vendor
```
2. 修改了swagger文档后执行
```shell
# docs/gen.go
swag init -d ../internal/web/controllers --parseDependency -g api_v1.go -o ./
swag init -d ../internal/web/controllers --parseDependency -g api_tool.go -o ./
```