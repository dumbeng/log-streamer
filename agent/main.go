package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	clients  = make(map[*websocket.Conn]struct{}) // 连接到WebSocket的客户端集合
	upgrader = websocket.Upgrader{}               // 用于将HTTP连接升级到WebSocket协议的Upgrader
	lock     sync.RWMutex
)

// handleConnections 处理WebSocket请求
func handleConnections(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true } // 允许跨域
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("ws connection failed, error: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer ws.Close()

	lock.Lock()
	clients[ws] = struct{}{}
	lock.Unlock()

	log.Infof("new connection from %s", r.RemoteAddr)

	for {
		// 保持连接活跃
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Printf("connection from %s stopped, error: %v", ws.RemoteAddr(), err)
			lock.Lock()
			delete(clients, ws)
			lock.Unlock()
			break
		}
	}
}

// handleMessages 监听日志文件变化并将更新发送给客户端
func handleMessages(ctx context.Context, logFilePath string, scanInterval int) {
	ticker := time.NewTicker(time.Duration(scanInterval) * time.Millisecond)
	defer ticker.Stop()

	file, err := os.Open(logFilePath)
	if err != nil {
		log.Errorf("failed to read log file, error: %v", err)
		return
	}

	stat, err := file.Stat()
	if err != nil {
		log.Errorf("failed to read file size, error: %v", err)
		_ = file.Close()
		return
	}

	lastSize := stat.Size()
	_ = file.Close()

	for {
		select {
		case <-ctx.Done():
			log.Info("file listener exit")
			return
		case <-ticker.C:
			file, err := os.Open(logFilePath)
			if err != nil {
				log.Errorf("failed to read log file, error: %v", err)
				continue
			}

			// 获取当前文件大小
			stat, err := file.Stat()
			if err != nil {
				log.Errorf("failed to read file size, error: %v", err)
				_ = file.Close()
				continue
			}

			offset := int64(0)
			currentSize := stat.Size()
			if currentSize == lastSize {
				// no new content
				log.Infof("no new content")
				_ = file.Close()
				continue
			} else if currentSize < lastSize {
				// log file rotated, need to restart
				offset = 0
			} else {
				offset = lastSize
			}

			log.Debugf("current offset: %d", offset)
			log.Debugf("current file size: %d", stat.Size())

			_, err = file.Seek(offset, 0)
			if err != nil {
				log.Errorf("failed to seek file at offset %d, error: %v", offset, err)
				_ = file.Close()
				continue
			}

			reader := bufio.NewReader(file)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					log.Infof("file read done: %v", err)
					break // 文件读取完毕
				}

				lock.RLock()

				var wg sync.WaitGroup
				wg.Add(len(clients))
				for client, _ := range clients {
					go func(client *websocket.Conn, line []byte) {
						err := client.WriteMessage(websocket.TextMessage, line)
						if err != nil {
							log.Errorf("failed to send message to client %s, error: %v", client.RemoteAddr(), err)
						}
						wg.Done()
					}(client, []byte(line))
				}
				wg.Wait()

				lock.RUnlock()
			}

			offset = currentSize

			_ = file.Close()
		}
	}
}

func main() {
	// 设置日志文件路径
	var logFilePath string
	var host string
	var port string
	var scanInterval int

	flag.StringVar(&logFilePath, "logfile", "./log.log", "Path to the log file")
	flag.StringVar(&host, "host", "127.0.0.1", "Host to listen")
	flag.StringVar(&port, "port", "9211", "Port to listen")
	flag.IntVar(&scanInterval, "interval", 500, "Interval to scan the file")
	flag.Parse()

	// 解析相对路径和绝对路径
	absPath, err := filepath.Abs(logFilePath)
	if err != nil {
		log.Fatal(err)
	}
	logFilePath = absPath

	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		FullTimestamp:    true,
		TimestampFormat:  "2006-01-02 15:04:05.233",
	})

	// 启动WebSocket服务
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go handleMessages(ctx, logFilePath, scanInterval)

	// 设置路由
	http.HandleFunc("/ws", handleConnections)

	// 启动HTTP服务器
	log.Printf("HTTP server started on %s:%s", host, port)
	err = http.ListenAndServe(fmt.Sprintf("%s:%s", host, port), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
