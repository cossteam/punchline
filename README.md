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

### 赋予权限
```
chmod +x punchline
```

### 服务器

```sh
./punchline server
```

### 客户端

```sh
./punchline client -c config/example-client.yaml
```

## ice模式

### 服务器

```sh
./punchline signal
```

### 客户端1
```sh
./punchline client --hostname client1 -subscriptions client2 --signalServer signalServer:7777
```

### 客户端2
```sh
./punchline client --hostname client2 -subscriptions client1 --signalServer signalServer:7777
```