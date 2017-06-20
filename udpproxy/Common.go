// message
/*
解析码流和构造码流
*/
package udpproxy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
)

//接收数据
type RecvData struct {
	DeviceAddr *net.UDPAddr //设备远端地址
	RspPort    int          //响应端口
	Data       []byte       //报文长度
}

//代理数据结构
type ProxyData struct {
	msg_type int          //消息类型 0-正常报文  1-连接释放
	peer     *net.UDPAddr //对端地址
	src_port int          //指定发送源端口 0-65535
	rsp_port int          //指定响应源端口 0-65535
	data     []byte       //净荷数据，定义为byte流
}

//获取接收地址关键字，对端的IP和端口 + 本端的接收端口
func (proxy_data *ProxyData) GetRspAddr() string {
	return proxy_data.peer.String() + ":" + strconv.Itoa(proxy_data.rsp_port)
}

//获取端口映射字符串
func (proxy_data *ProxyData) GetPorts() string {
	return strconv.Itoa(int(proxy_data.src_port)) + "-" + strconv.Itoa(int(proxy_data.rsp_port))
}

//从缓存中创建对象，采用大端编码方式
func BytesToProxyData(buf []byte) *ProxyData {
	//长度不够
	if len(buf) <= 2 || len(buf) > MAX_PKG_SIZE+12 {
		return nil
	}
	proxy_data := new(ProxyData)

	//消息类型 1字节
	proxy_data.msg_type = int(binary.BigEndian.Uint16(buf[0:2]))
	//查询消息可以直接返回
	if proxy_data.msg_type == MSG_QUERY {
		return proxy_data
	}
	//IP数据 4字节  端口 3*2=6字节
	proxy_data.peer = &net.UDPAddr{IP: net.IPv4(buf[2], buf[3], buf[4], buf[5]),
		Port: int(binary.BigEndian.Uint16(buf[6:8]))}
	proxy_data.src_port = int(binary.BigEndian.Uint16(buf[8:10]))
	proxy_data.rsp_port = int(binary.BigEndian.Uint16(buf[10:12]))
	//净荷
	length := int(len(buf) - 12)
	//	if DEBUG {
	//		fmt.Println(length, buf)
	//	}

	//更新data
	for i := 0; i < length; i++ {
		proxy_data.data = append(proxy_data.data, buf[12+i])
	}

	return proxy_data
}

//构造代理请求消息
func BuildReqMsg(msg_type int, udpaddr string, src_port int, rsp_port int, data []byte) *ProxyData {
	proxy_data := new(ProxyData)

	//消息类型 1字节
	proxy_data.msg_type = msg_type
	//查询消息可以直接返回
	if proxy_data.msg_type == MSG_QUERY {
		return proxy_data
	}
	//IP数据 4字节  端口 3*2=6字节
	var err error
	proxy_data.peer, err = net.ResolveUDPAddr("udp4", udpaddr)
	if err != nil {
		fmt.Println("IP非法", udpaddr, err.Error())
	}
	proxy_data.src_port = src_port
	proxy_data.rsp_port = rsp_port
	//净荷
	length := len(data)
	if DEBUG {
		fmt.Println(length, data)
	}

	//更新data
	for i := 0; i < length; i++ {
		proxy_data.data = append(proxy_data.data, data[i])
	}

	return proxy_data
}

//构造响应消息
func BuildRspMsg(addr *net.UDPAddr, data []byte) []byte {
	var buf bytes.Buffer
	//转换成IPv4
	buf.Write(addr.IP.To4())
	//获取端口
	port := make([]byte, 2)
	binary.BigEndian.PutUint16(port, uint16(addr.Port))
	buf.Write(port)
	//净荷数据
	buf.Write(data)
	return buf.Bytes()
}

//UDP发送
func UDPSend(conn *net.UDPConn, peer *net.UDPAddr, data []byte) {
	if conn == nil {
		//地址无效，不能拨号
		if peer == nil {
			return
		}
		//	新建连接发送
		conn_temp, err := net.Dial("udp", peer.String())
		if err != nil {
			fmt.Println("转发本地端口失败", peer.String())
			return
		}
		defer conn_temp.Close()
		conn_temp.Write(data)
	} else {
		conn.WriteToUDP(data, peer)
	}

}
