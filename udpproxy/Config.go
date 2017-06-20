// Config
package udpproxy

//测试模式
var DEBUG bool = false

//队列操作超时定时器
const TIMEOUT_QUEUE int = 1

//队列长度
var MAX_QUEUE_SIZE int = 512 * 100

//数据最大缓存
const MAX_PKG_SIZE int = 1500

//消息类型

const MSG_SEND = 0  //发送
const MSG_CLOSE = 1 //关闭
const MSG_QUERY = 2 //查询管理

//路由老化超时时间，单位s
var ROUTE_ALIVE int = 30

//心跳时间，接收和发送端口检查数据间隔时间，单位s
const TIMEOUT_HEART int = 0

//数据包接收等待定时器
var TIMEOUT_RECV int = 5
