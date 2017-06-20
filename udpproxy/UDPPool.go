// UDPPool
package udpproxy

import (
	queue "Proxy/queue"
	"bytes"
	//	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"sync"
)

//UDP隧道池
type UDPPool struct {
	lock   *sync.Mutex //线程池锁,注意，只针对dict读写锁，不能包括route，route自带锁，否则可能导致死锁
	dict   map[string]*UDPTunnel
	route  *RouteTable
	conn   *net.UDPConn //响应请求的UDP连接
	enable bool
}

//创建新的UDP隧道池
func NewUDPPool() *UDPPool {
	udppool := new(UDPPool)
	udppool.lock = new(sync.Mutex)
	udppool.dict = make(map[string]*UDPTunnel)
	udppool.route = newRouteTabel()
	udppool.enable = true
	return udppool
}

//更新UDP连接
func (udppool *UDPPool) SetConn(conn *net.UDPConn) {
	udppool.lock.Lock()
	defer udppool.lock.Unlock()
	udppool.conn = conn
}

//使用代理端口发送数据
func (udppool *UDPPool) Response(peer *net.UDPAddr, data []byte) {
	udppool.conn.WriteToUDP(data, peer)
}

//更新代理隧道，如果转发隧道已经存在，则直接返回
func (udppool *UDPPool) updateTunnel(key string, proxy_data *ProxyData) bool {
	//如果udp发送隧道未建立，则需要建立，发送隧道需要添加锁操作
	udppool.lock.Lock()
	defer udppool.lock.Unlock()
	if _, ok := udppool.dict[key]; !ok {

		if DEBUG {
			fmt.Println("新建转发通道：", key)
		}
		//新建转发通道
		udp := newUDPTunnel(queue.New(MAX_QUEUE_SIZE), queue.New(MAX_QUEUE_SIZE), proxy_data.src_port, proxy_data.rsp_port)
		if udp == nil {
			fmt.Println("创建转发通道失败")
			return false
		}
		udppool.dict[key] = udp

		//监听接收队列
		go udppool.listen(key)
	}

	return true
}

//新增转发端口
func (udppool *UDPPool) add(proxy_data *ProxyData, peer *net.UDPAddr) bool {

	key := proxy_data.GetPorts()

	//添加或者更新路由表，路由表自带锁操作
	if !udppool.route.Add(proxy_data.GetRspAddr(), peer) {
		return false
	}

	//更新UDP转发通道
	if !udppool.updateTunnel(key, proxy_data) {
		return false
	}

	//发送到队列
	err_put := udppool.dict[key].queue_send.Put(*proxy_data, float64(TIMEOUT_QUEUE))
	if err_put != nil {
		fmt.Println("转发数据入队异常：", err_put.Error(), key)
		return false
	}

	if DEBUG {
		fmt.Println("转发数据入队成功", key, proxy_data.GetRspAddr(), peer.String())
	}

	return true
}

//获取转发隧道状态，加锁访问
func (udppool *UDPPool) check(key string) bool {
	udppool.lock.Lock()
	defer udppool.lock.Unlock()
	//key存在且运行状态为正常
	if _, ok := udppool.dict[key]; ok {
		//如果运行不正常，删除路由
		if !udppool.dict[key].enable {
			delete(udppool.dict, key)
			return false
		} else {
			return true
		}
	} else {
		return false
	}
}

//监听和转发
func (udppool *UDPPool) listen(key string) {
	for udppool.enable {
		//判断udp转发端口池中是否存在转发隧道
		if udppool.check(key) {
			val, err := udppool.dict[key].queue_recv.Get(float64(TIMEOUT_HEART))
			if err != nil {
				if DEBUG {
					fmt.Println("数据分析心跳：", key)
				}
				continue
			}

			//查找路由表
			rspAddr := val.(RecvData).DeviceAddr.String() + ":" +
				strconv.Itoa(val.(RecvData).RspPort)
			peer := udppool.route.Get(rspAddr)
			if peer == nil {
				continue
			}

			udppool.Response(peer, val.(RecvData).Data)

		} else {
			fmt.Println("数据分析停止：", key)
			return
		}
	}

	fmt.Println("数据分析停止：", key)
}

//删除路由表，并检查端口是否空闲
func (udppool *UDPPool) rmv(proxy_data *ProxyData) {
	udppool.lock.Lock()
	defer udppool.lock.Unlock()
	//判断路由表中该port是否清空
	udppool.route.Rmv(proxy_data.GetRspAddr())
	//如果路由表空了，表示该端口不需要再监听，暂时不停止监听
	//	if udppool.route.IsEmpty(proxy_data.rsp_port) {
	//		//如果该端口存在，则去使能端口，从字典中删除
	//		udp, ok := udppool.dict[proxy_data.GetPorts()]
	//		if ok {
	//			udp.enable = false
	//			delete(udppool.dict, proxy_data.GetPorts())
	//			fmt.Println("关闭端口", proxy_data.GetPorts())
	//		}

	//	}

}

func (udppool *UDPPool) GetAll() []byte {
	var buf bytes.Buffer
	//查询隧道开放的端口
	buf.WriteString("Port:")
	//先获取锁
	udppool.lock.Lock()
	defer udppool.lock.Unlock()
	for k, _ := range udppool.dict {
		buf.WriteString(k)
		buf.WriteString(";")
	}
	return buf.Bytes()
}

//查询，返回查询结果
func (udppool *UDPPool) query() []byte {
	var buf bytes.Buffer
	//查询隧道开放的端口
	buf.Write(udppool.GetAll())
	//查询当前路由
	buf.WriteString("\nRoute:")
	buf.Write(udppool.route.GetAll())
	return buf.Bytes()
}

//UDP转发线程池处理
func (udppool *UDPPool) Send(proxy_data *ProxyData, raddr *net.UDPAddr) {
	if proxy_data == nil {
		fmt.Println("需要转发的数据端口非法：", proxy_data.peer.String())
		return
	}

	switch proxy_data.msg_type {
	//关闭端口
	case MSG_CLOSE:
		go udppool.rmv(proxy_data)
	//发送消息
	case MSG_SEND:
		//异步处理
		go udppool.add(proxy_data, raddr)
	//查询消息
	case MSG_QUERY:
		go func() {

			//			peer, _ := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
			if DEBUG {
				fmt.Println("处理查询请求：", raddr.String())
			}
			udppool.Response(raddr, BuildRspMsg(raddr, udppool.query()))
		}()
	default:
		fmt.Println("未知的消息类型：", proxy_data.msg_type)
	}

}
