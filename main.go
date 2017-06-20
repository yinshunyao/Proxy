// main
package main

import (
	udp "Proxy/udpproxy"
	"flag"
	"fmt"
)

func main() {
	//获取入参端口
	port := flag.Int("port", 61111, "代理服务监听的端口，默认使用61111")
	cache := flag.Int("cache", 512*100, "一个发送端口上缓存的最大数据包个数")
	age := flag.Int("age", 30, "路由老化时间，单位秒")
	timeout := flag.Int("timeout", 5, "接收数据包超时定时器")
	debug := flag.Bool("debug", false, "携带该参数将启动调试模式，增多打印，但性能将降低")
	flag.Parse()
	fmt.Println("[输入-h查看帮助]")
	udp.DEBUG = *debug
	udp.MAX_QUEUE_SIZE = *cache
	udp.ROUTE_ALIVE = *age
	udp.TIMEOUT_RECV = *timeout
	udp.UDPProxyListen(*port)
}
