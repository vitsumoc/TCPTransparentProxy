package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

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

	// 保持主程序运行
	select {}
}

func handleConnection(conn net.Conn, toServer string) {
	fmt.Println("接受客户端链接 ok")
	// 设置连接超时
	conn.SetDeadline(time.Now().Add(5 * time.Minute))
	// 确保连接关闭
	defer conn.Close()

	// 建立TCP连接，设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dialer := net.Dialer{}
	serverConn, err := dialer.DialContext(ctx, "tcp", toServer)
	if err != nil {
		fmt.Println("建立TCP连接出错:", err)
		return
	}

	// 设置服务器连接超时
	serverConn.SetDeadline(time.Now().Add(5 * time.Minute))
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
		New: func() any {
			b := make([]byte, 576)
			return &b
		},
	} // small buff pool
	LPool = sync.Pool{
		New: func() any {
			b := make([]byte, 64*1024+262)
			return &b
		},
	} // large buff pool for udp
)

func Transport(rw1, rw2 io.ReadWriter) error {
	errc := make(chan error, 2) // 改为2，等待两个goroutine都完成

	go func() {
		b := LPool.Get().(*[]byte)
		defer LPool.Put(b)

		_, err := io.CopyBuffer(rw1, rw2, *b)
		errc <- err
	}()

	go func() {
		b := LPool.Get().(*[]byte)
		defer LPool.Put(b)

		_, err := io.CopyBuffer(rw2, rw1, *b)
		errc <- err
	}()

	// 等待两个goroutine都完成
	err1 := <-errc
	err2 := <-errc

	// 如果两个都正常结束或EOF，返回nil
	if (err1 == nil || err1 == io.EOF) && (err2 == nil || err2 == io.EOF) {
		return nil
	}

	// 返回非EOF的错误
	if err1 != nil && err1 != io.EOF {
		return err1
	}
	if err2 != nil && err2 != io.EOF {
		return err2
	}

	return nil
}
