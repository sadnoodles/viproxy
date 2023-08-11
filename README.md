VIProxy
=======

原始项目来源：

https://github.com/brave/viproxy

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
`go build -o vsock_proxy example/vsock_proxy.go`

也可以直接下载bin/目录下的编译文件，移动到可执行文件目录：

```
mkdir -p /usr/local/bin/vsock_proxy/
mv vsock_proxy /usr/local/bin/vsock_proxy/vsock_proxy
chmod +x /usr/local/bin/vsock_proxy/vsock_proxy
```

1. 宿主机上运行：
`VSOCK_INADDRS=0:8000 VSOCK_OUTADDRS=127.0.0.1:8000 /usr/local/bin/vsock_proxy/vsock_proxy -a install`

2. 虚拟机上运行：
`VSOCK_INADDRS=127.0.0.1:8000 VSOCK_OUTADDRS=1:8000 /usr/local/bin/vsock_proxy/vsock_proxy -a install`

多个地址同时监听，仅运行：
`VSOCK_INADDRS=127.0.0.1:8080,127.0.0.1:8081 VSOCK_OUTADDRS=1:8080,1:8081 ./vsock_proxy`

宿主机、虚拟机上查看服务状态：

`systemctl status vsock_proxy`

查看日志：

`journalctl -xe -u vsock_proxy`

查看配置文件：
```
ls /usr/local/bin/vsock_proxy/
cat /usr/local/bin/vsock_proxy/vsock_proxy_config.json
```

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

宿主机访问虚拟机则IN/OUT反转，需要知道虚拟机的CID。

4. CID 测试

编译
`/usr/local/bin/vsock_proxy/vsock_proxy -c`

也可直接使用`bin/cid`测试

5. 完全卸载服务

```
/usr/local/bin/vsock_proxy/vsock_proxy -a stop
/usr/local/bin/vsock_proxy/vsock_proxy -a uninstall
rm -rf /usr/local/bin/vsock_proxy/
```
