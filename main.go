package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

type Tconfig struct {
	Proxy []struct {
		From string `yaml:"from"`
		To   string `yaml:"to"`
	}
}

func main() {
	// 读取配置文件
	confContent, err := os.ReadFile("./config.yml")
	if err != nil {
		log.Fatal(err.Error())
	}

	confOb := Tconfig{}
	err = yaml.Unmarshal([]byte(confContent), &confOb)
	if err != nil {
		log.Fatal(err.Error())
	}

	// 启动每一组代理
	for x := 0; x < len(confOb.Proxy); x++ {
		go func(x int) {
			// 监听端口
			ln, err := net.Listen("tcp", confOb.Proxy[x].From)
			if err != nil {
				fmt.Println("监听端口出错:", err)
				return
			}
			// 确保监听端口关闭
			defer ln.Close()

			fmt.Println("服务器已启动，正在监听" + confOb.Proxy[x].From)
			for {
				// 接受客户端连接
				conn, err := ln.Accept()
				if err != nil {
					fmt.Println("接受客户端连接出错:", err)
					continue
				}
				// 处理客户端连接
				go handleConnection(conn, confOb.Proxy[x].To)
			}
		}(x)
	}

	for {
	}
}

func handleConnection(conn net.Conn, toServer string) {
	fmt.Println("接受客户端链接 ok")
	// 确保连接关闭
	defer conn.Close()

	// 解析服务器地址
	serverAddr, err := net.ResolveTCPAddr("tcp", toServer)
	if err != nil {
		fmt.Println("解析服务器地址出错:", err)
		return
	}
	// 建立TCP连接
	serverConn, err := net.DialTCP("tcp", nil, serverAddr)
	if err != nil {
		fmt.Println("建立TCP连接出错:", err)
		return
	}
	fmt.Println("到服务器建链 ok")
	// 确保连接关闭
	defer serverConn.Close()

	// 把两边数据对接上
	err = Transport(serverConn, conn)
	if err == nil {
		fmt.Println("链接正常断开")
	}
	if err != nil {
		fmt.Println(err)
		return
	}
}

// buffer pools
var (
	SPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 576)
		},
	} // small buff pool
	LPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 64*1024+262)
		},
	} // large buff pool for udp
)

func Transport(rw1, rw2 io.ReadWriter) error {
	errc := make(chan error, 1)
	go func() {
		b := LPool.Get().([]byte)
		defer LPool.Put(b)

		_, err := io.CopyBuffer(rw1, rw2, b)
		errc <- err
	}()

	go func() {
		b := LPool.Get().([]byte)
		defer LPool.Put(b)

		_, err := io.CopyBuffer(rw2, rw1, b)
		errc <- err
	}()

	err := <-errc
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}
