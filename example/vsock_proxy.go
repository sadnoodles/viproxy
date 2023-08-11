package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"net"
	"strconv"
	"strings"

	"github.com/brave/viproxy"
	"github.com/kardianos/service"
	"github.com/mdlayher/vsock"
	"github.com/spf13/viper"
)

var BuildTime = ""
var BuildCommit = ""

type VsockProxyConfig struct {
	InAddrs  string `json:"inaddrs"`
	OutAddrs string `json:"outaddrs"`
}

func parseAddr(rawAddr string) net.Addr {
	var addr net.Addr
	var err error

	addr, err = net.ResolveTCPAddr("tcp", rawAddr)
	if err == nil {
		return addr
	}

	// We couldn't parse the address, so we must be dealing with AF_VSOCK.  We
	// expect an address like 3:8080.
	fields := strings.Split(rawAddr, ":")
	if len(fields) != 2 {
		log.Fatal("Looks like we're given neither AF_INET nor AF_VSOCK addr.")
	}
	cid, err := strconv.ParseInt(fields[0], 10, 32)
	if err != nil {
		log.Fatal("Couldn't turn CID into integer.")
	}
	port, err := strconv.ParseInt(fields[1], 10, 32)
	if err != nil {
		log.Fatal("Couldn't turn port into integer.")
	}

	addr = &vsock.Addr{ContextID: uint32(cid), Port: uint32(port)}

	return addr
}

func (proxy *VsockProxyConfig) Start(ctx context.Context, logger service.Logger) error {

	// E.g.: VSOCK_INADDRS=127.0.0.1:8080,127.0.0.1:8081 VSOCK_OUTADDRS=4:8080,4:8081 go run main.go
	inEnv, outEnv := proxy.InAddrs, proxy.OutAddrs
	if inEnv == "" || outEnv == "" {
		log.Fatal("Environment variables VSOCK_INADDRS and VSOCK_OUTADDRS not set.")
	}

	rawInAddrs, rawOutAddrs := strings.Split(inEnv, ","), strings.Split(outEnv, ",")
	if len(rawInAddrs) != len(rawOutAddrs) {
		log.Fatal("VSOCK_INADDRS and VSOCK_OUTADDRS must contain same number of addresses.")
	}

	var tuples []*viproxy.Tuple
	for i := range rawInAddrs {
		inAddr := parseAddr(rawInAddrs[i])
		outAddr := parseAddr(rawOutAddrs[i])
		tuples = append(tuples, &viproxy.Tuple{InAddr: inAddr, OutAddr: outAddr})
	}

	p := viproxy.NewVIProxy(tuples)
	if err := p.Start(); err != nil {
		log.Fatalf("Failed to start VIProxy: %s", err)
	}
	<-make(chan bool)
	return nil
}

type Program struct {
	config *VsockProxyConfig
	ctx    context.Context
	cancel context.CancelFunc
}

func NewProgram(saveconf bool) (*Program, error) {
	viper.SetConfigName("vsock_proxy_config")
	viper.SetConfigType("json")
	if binfile, err := os.Executable(); err == nil {
		if binpath, err := filepath.Abs(filepath.Dir(binfile)); err == nil {
			viper.AddConfigPath(binpath)
		}
	}
	viper.AddConfigPath(".")
	viper.SetEnvPrefix("VSOCK")
	viper.BindEnv("inaddrs")
	viper.BindEnv("outaddrs")
	viper.AutomaticEnv()
	// 尝试从文件读取配置
	viper.ReadInConfig()
	// 解析配置
	var config VsockProxyConfig
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	if viper.ConfigFileUsed() == "" && saveconf {
		// 配置从环境变量种加载时
		viper.SafeWriteConfig()
	} else {
		viper.WriteConfig()
	}

	// FIXME: cancel后，bot有机会提交记录吗
	ctx, cancel := context.WithCancel(context.Background())
	return &Program{config: &config, ctx: ctx, cancel: cancel}, nil
}

func (p *Program) Start(s service.Service) (err error) {
	var logger service.Logger = service.ConsoleLogger
	if runtime.GOOS != "darwin" {
		logger, err = s.Logger(nil)
		if err != nil {
			return err
		}
	}
	err = p.config.Start(p.ctx, logger)
	if err != nil {
		return err
	}
	return nil
}

func (p *Program) Stop(s service.Service) error {
	p.cancel()
	return nil
}

func GetCid() {

	cid, err := vsock.ContextID()
	var s string
	if err == nil {
		s = fmt.Sprint(cid)
	}

	fmt.Printf("CID is: %s\n", s)

}

func main() {
	action := flag.String("a", "", "action: start/restart/stop/status/install/uninstall")
	cid := flag.Bool("c", false, "Print current CID.")
	help := flag.Bool("h", false, "help")
	flag.Parse()
	if *help {
		fmt.Println("BuildTime:", BuildTime, "BuildCommit", BuildCommit)
		fmt.Println("Usage: vsock_proxy -c  Print current CID.")
		fmt.Println("       vsock_proxy -a [start/restart/stop/status/install/uninstall]")
		fmt.Println(`
· start: 启动服务
· restart: 重启服务
· stop: 停止服务
· status: 服务状态
· install: 安装服务
· uninstall: 卸载服务`)
		return
	}
	if *cid {
		GetCid()
		os.Exit(0)
	}

	svcConfig := &service.Config{
		Name:        "vsock_proxy",
		DisplayName: "Vsock Proxy",
		Description: "Vsock Proxy: TCP <-> VSOCK, Host <-> VM.",
		Arguments:   []string{"-a", "daemon"},
		Option: service.KeyValue{
			"OnFailure":              "restart",
			"OnFailureDelayDuration": "10s",
			"OnFailureResetPeriod":   10,
			"DelayedAutoStart":       true,
			"StartType":              "automatic",
		},
	}

	saveconf := (*action != "")
	// 非服务模式不保存配置文件
	prg, err := NewProgram(saveconf)
	if err != nil {
		log.Fatal(err)
	}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	switch *action {
	case "start":
		if err = s.Start(); err != nil {
			log.Fatal(err)
		} else {
			log.Println(*action, "OK")
		}
	case "restart":
		if err = s.Restart(); err != nil {
			log.Fatal(err)
		} else {
			log.Println(*action, "OK")
		}
	case "stop":
		if err = s.Stop(); err != nil {
			log.Fatal(err)
		} else {
			log.Println(*action, "OK")
		}
	case "status":
		if status, err := s.Status(); err != nil {
			log.Fatal(err)
		} else {
			statusMap := map[service.Status]string{
				service.StatusUnknown: "StatusUnknown",
				service.StatusRunning: "StatusRunning",
				service.StatusStopped: "StatusStopped",
			}
			log.Println(statusMap[status])
		}
	case "install":
		if err = s.Install(); err != nil {
			log.Fatal(err)
		} else {
			log.Println(*action, "OK")
		}
		if err = s.Start(); err != nil {
			log.Fatal(err)
		} else {
			log.Println("start", "OK")
		}
	case "uninstall":
		if err = s.Uninstall(); err != nil {
			log.Fatal(err)
		} else {
			log.Println(*action, "OK")
		}
	case "daemon":
		if err = s.Run(); err != nil {
			log.Fatal(err)
		}
	default:
		err := prg.Start(s)
		if err != nil {
			log.Fatalln(err)
		}
		quit := make(chan os.Signal, 1)                      // 创建一个 os.Signal 类型的 Channel
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // 监听关闭信号，Ctrl+C 或者其他情况关闭进程都会触发
		<-quit                                               // 收到关闭信号前挂起，收到信号后执行后面的代码
		prg.cancel()
	}
}
