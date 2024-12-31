# TCPTransparentProxy

一个创建 TCP 透明代理的小工具

在 `config.yml` 中配置每一组透明代理的入口和访问目标：

```yml config.yml
proxy:
  - from: "0.0.0.0:19000"
    to: "192.168.1.13:9000"
  - from: "0.0.0.0:19001"
    to: "192.168.1.13:102"

```

将可执行程序和配置文件放在同一路径，运行程序即可。
