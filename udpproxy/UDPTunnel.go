// UDPTunnel
package udpproxy

import (
	//	reuse "Proxy/go-reuseport-master"
	queue "Proxy/queue"
	"fmt"
	"net"
	//	"strconv"
	//	"sync"
	"time"
)

/*UDP隧道结构体*/
type UDPTunnel struct {
	queue_send *queue.Queue //发送队列
	queue_recv *queue.Queue //接收队列
	src_port   int
	rsp_port   int
	//	send_conn  *net.UDPConn   //发送UDP连接
	//	rsp_conn net.PacketConn //接收UDP连接
	enable bool //是否有效
}

//创建新的UDP代理连接
func newUDPTunnel(queue_send *queue.Queue, queue_recv *queue.Queue, src_port int, rsp_port int) *UDPTunnel {
	udp := new(UDPTunnel)
	udp.queue_send = queue_send
	udp.queue_recv = queue_recv
	udp.src_port = src_port
	udp.rsp_port = rsp_port
	udp.enable = true

	if !udp.duplex() {
		return nil
	}

	return udp
}

func (udp *UDPTunnel) duplex() bool {
	conn_recv, err := net.ListenUDP("udp4", &net.UDPAddr{Port: udp.rsp_port})
	if err != nil {
		fmt.Println("双工模式端口开放失败", udp.rsp_port)
		return false
	}

	var conn_send *net.UDPConn
	if udp.src_port == udp.rsp_port {
		conn_send = conn_recv
	} else {
		conn_send, err = net.ListenUDP("udp4", &net.UDPAddr{Port: udp.src_port})
		if err != nil {
			fmt.Println("双工模式端口开放失败", udp.src_port)
			return false
		}

	}

	go udp.recvFromUDP(conn_recv)
	go udp.sendToUDP(conn_send)
	return true
}

/*发送队列监听*/
func (udp *UDPTunnel) sendToUDP(conn *net.UDPConn) {
	defer conn.Close()
	for udp.enable {
		//从队列中获取需要发送的数据
		val, err := udp.queue_send.Get(float64(TIMEOUT_HEART))
		if err != nil {
			if DEBUG {
				fmt.Println("发送端口心跳：", udp.src_port)
			}
			continue
		}

		if val.(ProxyData).msg_type == 1 {
			udp.enable = false
			return
		}

		if DEBUG {
			fmt.Println("需要转发数据：", val.(ProxyData).peer, val.(ProxyData).data)
		}

		//发送数据
		_, err_send := conn.WriteToUDP(val.(ProxyData).data, val.(ProxyData).peer) //c.Write(proxyData.data)
		if err_send != nil {
			fmt.Println("转发UDP数据发送异常：", err_send.Error())
		} else {
			if DEBUG {
				fmt.Println("转发数据成功：", val.(ProxyData).peer.String(), val.(ProxyData).data)
			}
		}
	}

	fmt.Println("发送端口关闭:", udp.src_port)

}

/*UDP端口接收监听*/
func (udp *UDPTunnel) recvFromUDP(conn *net.UDPConn) {
	// 退出时关闭连接
	defer conn.Close()
	//初始化缓存
	var buf [MAX_PKG_SIZE]byte
	fmt.Println("开始监听端口：", conn.LocalAddr().String())

	// 处理接收到的消息
	for udp.enable {
		// 设置超时定时器，心跳
		if TIMEOUT_HEART > 0 {
			if err := conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(TIMEOUT_HEART))); err != nil {
				fmt.Println("设置接收定时器异常：", err.Error())
				time.Sleep(time.Second * 5)
				continue
			}

		}

		n, rAddr, err := conn.ReadFromUDP(buf[0:])
		if err != nil {
			if DEBUG {
				fmt.Println("接收端口异常:", conn.LocalAddr().String(), err.Error())
			}
			continue
		}

		//转换接收到的数据
		recv := RecvData{RspPort: udp.rsp_port, DeviceAddr: rAddr, Data: BuildRspMsg(rAddr, buf[:n])}
		if DEBUG {
			fmt.Println("接收到数据：", recv.DeviceAddr.String(), n)
		}

		//将接收到的数据放入到队列中
		err_put := udp.queue_recv.Put(recv, float64(TIMEOUT_QUEUE))
		if err_put != nil {
			fmt.Println("接收缓存队列异常，可能丢失消息", err_put.Error())
			continue
		}
	}

	fmt.Println("接收端口关闭：", conn.LocalAddr().String())
}
