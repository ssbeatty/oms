## 项目简介

​本项目使用beego和ssh、sftp开发。主要用来在网络条件差或者没有其他可用的ssh客户端时做批量任务执行和ssh连接。目的是做一个阉割版的xshell并简单的实现ansible或者saltstack的部分功能。

#### 项目依赖
https://github.com/astaxie/beego

https://github.com/gorilla/websocket

#### 目前已经实现的功能

1. 基于websocket和xtermjs的webssh。
![image](./images/ssh.png)
2. 文件的批量分发。
![image](./images/file.png)
3. 批量执行命令。
![image](./images/shell.png)
4. 主机文件的浏览和下载。
![image](./images/browse.png)
5. 主机的curd管理。
![image](./images/hosts.png)
6. 使用组和标签的匹配以及模仿saltstack的匹配。
```shell script
-L 列表匹配
-L "192.168.1.1,192.168.1.2"
-E 正则
-E ".*？"
-G 通配符
-L "192.168.1.*"
以上匹配都只针对主机的addr
```
7. 后台定时任务判断主机的状态和清理没用的密钥。

#### 目前正在开发的

1. 文件浏览器的上传文件删除文件
2. 数据库内容的导入导出
3. ~~密钥存为文本~~

## 感谢
https://github.com/mojocn/felix

https://github.com/pytool/ssh

https://github.com/hequan2017/go-webssh

## 最后
因为时间紧促且自己学习go时间比较短, 所以代码写的比较烂, 之后会持续开发修复BUG, 如果感兴趣的可以给个star, 也可联系我共同学习

email & qq: 918562230@qq.com