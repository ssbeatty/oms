## 项目简介
本项目使用gin、gorm和ssh、sftp开发。旨在编写一个轻量，易用，多平台的运维项目。

现阶段目的是做一个阉割版的xshell并简单的实现ansible或者saltstack的部分功能。

**API文档**

[markdown](./api/v1/README.md)    [swagger](./api/v1/swagger.json)

#### 使用说明
1. 安装编译
```shell script
# 安装packr工具 需要go 1.16以上
go install github.com/gobuffalo/packr/packr@latest

# clone
git clone --recurse-submodules https://github.com/ssbeatty/oms.git

# build frontend
cd web/omsUI
yarn && yarn build

# 打包 oms
# linux
packr build -o oms cmd/omsd/main.go
# win
packr build -o oms.exe cmd/omsd/main.go
```

2. 启动 创建config.yaml在可执行文件同级 运行即可
```shell script
# configs/config.yaml.example
# 支持mysql postgres sqlite(默认, 仅调试)
app:
  name: oms
  addr: 127.0.0.1
  port: 8080
  mode: dev
  run_start: false # 是否在运行时打开浏览器 windows

db:
  driver: postgres
  user: root
  password: 123456
  dsn: 127.0.0.1:3306
  db_name: oms
```

3. 关于配置, 默认使用内嵌在二进制文件中的`configs/config.yaml.example`, 如果当前目录存在`config.yaml`则以此文件优先。

#### 目前已经实现的功能
1. web界面[omsUI](https://github.com/lixin59/omsUI/blob/master/README.md)
2. 隧道, 类似`ssh`的`-L`和`-R`
3. cron任务和长进程的管理
4. ssh命令批量执行
5. 文件批量的上传 流式传输支持大文件
6. 基于`sftp`文件浏览器
7. 基于novnc的vnc viewer