# testnode-pinger说明文档

## 概述

testnode-pinger 部署在被测试的机器上，构建了以下API服务，可在配置文件中修改路径：

    "/test/ips/region"    
    "/test/ips/tation"
    "/test/ips/results"
    "/test/ips/switchoff"
    "/test/ips/cleanup"

## 注意事项

1. ips初始化为空，此时不执行ping任务
2. 当同时接收到两种post的ips后，ping任务才开始执行，并将结果保存到results中。
3. 在ping任务运行时，通过接口总是能获取到上次的ping results.
4. 访问switchoff接口，可强制停止ping任务

## 快速开始

**注意：请使用root权限登录待测试服务器，否则可能导致pinger服务无法正常部署和运行！**

### 部署服务

```shell
wget https://github.com/wuguizhu/test/raw/master/shell/deploy.sh && . deploy.sh
```
