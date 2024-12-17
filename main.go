package main

import (
	"fmt"
	"io"
	"net"
	"sync"
)

// 从哪里代理到哪里
var importServer string = "0.0.0.0:19000"
var toServer string = "192.168.1.12:9000"

// 纯 54321 端口透明代理
func main() {
	// 监听端口
	ln, err := net.Listen("tcp", importServer)
	if err != nil {
		fmt.Println("监听端口出错:", err)
		return
	}
	// 确保监听端口关闭
	defer ln.Close()

	fmt.Println("服务器已启动，正在监听" + importServer)
	for {
		// 接受客户端连接
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("接受客户端连接出错:", err)
			continue
		}
		// 处理客户端连接
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	fmt.Println("客户端进来了")
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
	fmt.Println("服务器连上了")
	// 确保连接关闭
	defer serverConn.Close()

	// 把两边数据对接上
	err = Transport(serverConn, conn)
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

	if err := <-errc; err != nil && err != io.EOF {
		return err
	}
	return nil
}
