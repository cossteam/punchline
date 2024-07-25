# punchline
punchline一个简单的udppunch hole

## 概述

## 特性
- **UDP 通信：** 处理基于 UDP 的消息交换。
- **gRPC API：** 提供用于发布/订阅操作的 gRPC 接口。
- **动态主机管理：** 动态管理主机信息和通知。
- **插件支持：** 支持加载和管理客户端插件。

## 使用

### 构建项目：

```sh
make build
```

### 服务器

```sh
./punchline server
```

### 客户端

```sh
./punchline client -c config/example-client.yaml
```