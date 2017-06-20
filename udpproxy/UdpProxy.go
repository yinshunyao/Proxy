// UdpProxy
package udpproxy

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

//UDP代理
type UDPProxy struct {
	ListenPort int //监听端口
	//	queue_send *queue.Queue //代理端口发送数据
	conn *net.UDPConn
}

func UDPProxyListen(port int) bool {
	//合法端口
	if port <= 0 {
		port = 61111
	}

	fmt.Println("代理监听端口", port)

	udppool := NewUDPPool()

	//监听端口
	for {
		service := ":" + strconv.Itoa(port)
		udpAddr, err := net.ResolveUDPAddr("udp4", service)
		if err != nil {
			fmt.Println("绑定端口异常：", err.Error())
			time.Sleep(time.Second)
			continue
		}

		conn, err := net.ListenUDP("udp", udpAddr)
		defer conn.Close()
		if err != nil {
			fmt.Println("监听端口异常：", err.Error())
			time.Sleep(time.Second)
			continue
		}
		//设置当前UDP响应连接
		udppool.SetConn(conn)
		fmt.Println("代理监听运行：", port)
		//处理请求
		handle_proxy(conn, udppool)
		//如果处理异常，暂停1s
		time.Sleep(time.Second)
	}

	return false
}

//接收到转发请求的处理
func handle_proxy(conn *net.UDPConn, udppool *UDPPool) {
	//	defer conn.Close()
	var buf [MAX_PKG_SIZE]byte
	for {

		//接收到转发请求
		n, raddr, err := conn.ReadFromUDP(buf[0:])
		if err != nil {
			fmt.Println("监听代理请求过程发生异常:", err.Error())
			return
		}

		//解析数据
		data := BytesToProxyData(buf[0:n])
		udppool.Send(data, raddr)

	}

}
