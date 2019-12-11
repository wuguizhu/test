# testnode-pinger说明文档

## 概述

testnode-pinger 部署在被测试的机器上，构建了以下API服务，可在配置文件中修改路径：

    "/test/ips/region"    
    "/test/ips/tation"
    "/test/ips/results"
    "/test/ips/switchoff"
    "/test/ips/cleanup"

## 基本介绍

1. 服务启动后未接收到ip前，此时不执行ping任务
2. 当接收由master发送的ip后，ping任务自动开始执行
3. 在ping任务运行时，通过接口总是能获取到上次ping任务完成后的结果
4. 访问switchoff接口，可强制停止ping任务
5. 访问cleanup接口，可停止卸载服务，并清理产生的文件。

## 关于配置文件

服务配置文件路径: test-master/conf/app.conf
ping操作配置文件路径: test-master/conf/testnodeprober.json

默认情况下，无需修改配置文件，即可正常运行，但以下情况请注意：

1. 服务默认占用8043接口，若接口已经被占用，请更改app.conf中
    
    httpport = 8043 #改为其他未被占用的端口号

2. 待补充

## 快速开始

**注意：操作前，请切换到root用户，否则可能导致pinger服务无法正常部署和运行！**

### 部署服务

```shell
wget https://github.com/wuguizhu/test/raw/master/shell/deploy.sh && chmod 777 deploy.sh&&. deploy.sh
```

### 卸载服务并执行清理

使用任意浏览器访问以下url即可卸载并清理

    http://[替换为测试服务器ip]:8043/test/ips/cleanup

若您已经登录测试服务器，也可以使用以下命令进行卸载清理

```shell
wget https://github.com/wuguizhu/test/raw/master/shell/clean.sh && . clean.sh
```
或者

```shell
curl http://[替换为测试服务器ip]:8043/test/ips/cleanup
```
