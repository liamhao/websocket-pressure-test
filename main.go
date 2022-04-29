package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// 总连接数
	client_num = 400000
	// 网卡名称
	network_card_name = "eth0"
	// 每个新增的IP可分配的连接数
	per_vip_clinet_num = 50000
	// 新增的IP地址池
	vip_pool = []string{}
	// 新增的IP地址范围
	vip_area = "192.168.0."
	// 同子网下的ip才能被其他机器访问
	vip_start_addr = 200

	// 被测试的目标地址
	target_addr = "ws://192.168.0.161:31301"

	// 是否启用发送消息，测试带宽
	enable_send_msg = false
	// 发送消息的间隔时长，默认间隔一秒
	send_msg_interval = 1 * time.Second
	// 测试的消息内容
	send_msg_content = "一二三四五六七八九十"
)

// 启动连接
func connect(local_vip string) {

	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 3 * time.Minute,
	}

	// 端口号0,使系统随机分配端口
	netAddr, err := net.ResolveTCPAddr("tcp4", local_vip+":0")

	if err != nil {
		log.Printf("e,%v", err)
		return
	}

	// 绑定本机指定IP
	dialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		d := net.Dialer{
			LocalAddr: netAddr,
		}
		return d.DialContext(ctx, network, addr)
	}

	// 开始连接
	ws, _, err := dialer.Dial(target_addr, nil)

	if err != nil {
		log.Printf("dial:%v", err)
		return
	}

	defer ws.Close()

	if enable_send_msg {
		for {
			ws.WriteMessage(websocket.TextMessage, []byte(send_msg_content))
			time.Sleep(send_msg_interval)
		}
	}
}

// 根据需要创建多网卡
func createVipToEth0(count int) {
	for i := 1; i <= count; i++ {
		vip := vip_area + strconv.Itoa(vip_start_addr+i)
		cmd := exec.Command("sh", "-c", "ifconfig "+network_card_name+":"+strconv.Itoa(i)+" "+vip+" up")
		if _, err := cmd.Output(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		vip_pool = append(vip_pool, vip)
	}
}

// 销毁多网卡
func deleteVipToEth0() {
	for i := 1; i <= len(vip_pool); i++ {
		exec.Command("sh", "-c", "ifconfig "+network_card_name+":"+strconv.Itoa(i)+" "+vip_area+strconv.Itoa(vip_start_addr+i)+" down")
	}
}

func main() {

	count := int(math.Ceil(float64(client_num) / float64(per_vip_clinet_num)))

	createVipToEth0(count)

	defer deleteVipToEth0()

	for i := 1; i <= client_num; i++ {
		go connect(vip_pool[i%count])
	}

	select {}
}
