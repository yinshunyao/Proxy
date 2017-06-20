// RouteTable
package udpproxy

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

//路由
type Route struct {
	Peer       *net.UDPAddr //对端地址
	CreateTime int64        //创建时间，UNIX风格时间戳
}

//路由表
type RouteTable struct {
	lock   *sync.Mutex       //锁
	dict   map[string]*Route //字典
	enable bool
}

//新建路由表
func newRouteTabel() *RouteTable {
	table := new(RouteTable)
	table.lock = new(sync.Mutex)
	//初始化路由表字典为空
	table.dict = make(map[string]*Route)
	table.enable = true
	//启动检查线程
	go table.Check()
	return table
}

//增加路由
func (route *RouteTable) Add(rsp_addr string, peer *net.UDPAddr) bool {
	if rsp_addr == "" {
		fmt.Println("添加路由参数非法", rsp_addr, peer.String())
		return false
	}
	//获取锁
	route.lock.Lock()
	//解锁
	defer route.lock.Unlock()
	now := time.Now().Unix()
	//赋值，记录路由结构体的地址
	route.dict[rsp_addr] = &Route{Peer: peer, CreateTime: now}
	return true
}

//检查路由老化
func (route *RouteTable) Check() {
	for route.enable {
		//写锁
		route.lock.Lock()
		//		defer route.lock.Unlock()

		now := time.Now().Unix()
		//遍历字典，删除老化路由
		for k, v := range route.dict {
			//如果老化
			if now-v.CreateTime > int64(ROUTE_ALIVE) {
				delete(route.dict, k)
				if DEBUG {
					fmt.Println("路由老化：", k)
				}
			}
		}
		route.lock.Unlock()
		// 等待超时时间再全局检查一次
		time.Sleep(time.Second * time.Duration(ROUTE_ALIVE))
	}

}

//统计包含某端口的路由
func (route *RouteTable) IsEmpty(port int) bool {
	for k, _ := range route.dict {
		if strings.Contains(k, ":"+strconv.Itoa(port)) {
			return false
		}
	}

	return true
}

//删除路由表
func (route *RouteTable) Rmv(rsp_addr string) {
	//先判段对端地址是否存在
	peer := route.Get(rsp_addr)
	if peer != nil {
		route.lock.Lock()
		defer route.lock.Unlock()
		delete(route.dict, rsp_addr)
	}

}

//获取端口
func (route *RouteTable) Get(rsp_addr string) *net.UDPAddr {
	//先申请锁
	route.lock.Lock()
	defer route.lock.Unlock()
	if rsp_addr == "" {
		fmt.Println("查询参数非法")
		return nil
	}
	//
	route_info, err := route.dict[rsp_addr]
	if !err {
		fmt.Println("路由表中未查找到对应地址：", rsp_addr)
		return nil
	}

	//接收到回包则刷新路由时间
	route.dict[rsp_addr].CreateTime = time.Now().Unix()
	return route_info.Peer

}

//获取端口
func (route *RouteTable) GetAll() []byte {
	//先申请锁
	route.lock.Lock()
	defer route.lock.Unlock()
	var buf bytes.Buffer
	//遍历字典，获取路由
	for k, _ := range route.dict {
		buf.WriteString(k + ";")
	}
	return buf.Bytes()

}
