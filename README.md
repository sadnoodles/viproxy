VIProxy
=======

The VIProxy package implements a TCP proxy that translates between AF\_INET and
AF\_VSOCK connections.  The proxy takes as input two addresses, one being
AF\_INET and the other being AF\_VSOCK.  The proxy then starts a TCP listener on
the in-address and once it receives an incoming connection to the in-address, it
establishes a TCP connection to the out-addresses.  Once both connections are
established, the proxy copies data back and forth.

The [example](example) directory contains a simple example of how one would use
viproxy.

## 使用说明

编译
`go build -o vsock_proxy example/main.go`

也可以直接下载bin/目录下的编译文件

1. 宿主机上运行：
`IN_ADDRS=0:8000 OUT_ADDRS=127.0.0.1:8000 ./vsock_proxy`

2. 虚拟机上运行：
`IN_ADDRS=127.0.0.1:8000 OUT_ADDRS=1:8000 ./vsock_proxy`

3. 在虚拟机上测试：
`curl http://127.0.0.1:8000/`

其中特殊的CID：0代码宿主机，1代表local当前虚拟机。虚拟机自己的CID从3开始。

流量走向示意图：
```
curl-->127.0.0.1:8000(vm tcp) -> 1:8000(vm vsock) ---|--> 0:8000(hypervisor vsock) -> 127.0.0.1:8000(hypervisor tcp/http server)
                                                    |
                                                    |
                                          VM     虚拟化边界   宿主机
```
