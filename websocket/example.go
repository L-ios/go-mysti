package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocket 升级配置
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 生产环境应验证来源
	},
}

// 客户端结构
type Client struct {
	ID     string
	Conn   *websocket.Conn
	Send   chan []byte
	UserID string
}

// 消息结构
type Message struct {
	Type    string      `json:"type"`    // 消息类型: join, message, leave
	Content string      `json:"content"` // 消息内容
	UserID  string      `json:"user_id"` // 用户ID
	Time    int64       `json:"time"`    // 时间戳
	Data    interface{} `json:"data"`    // 扩展数据
}

// WebSocket 中心管理
type WebSocketHub struct {
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	mutex      sync.RWMutex
}

var hub = &WebSocketHub{
	Broadcast:  make(chan []byte),
	Register:   make(chan *Client),
	Unregister: make(chan *Client),
	Clients:    make(map[*Client]bool),
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mutex.Lock()
			h.Clients[client] = true
			h.mutex.Unlock()

			// 通知所有客户端有新用户加入
			msg := Message{
				Type:   "user_joined",
				UserID: client.UserID,
				Time:   time.Now().Unix(),
				Data:   map[string]interface{}{"client_id": client.ID},
			}
			msgBytes, _ := json.Marshal(msg)
			h.Broadcast <- msgBytes

		case client := <-h.Unregister:
			h.mutex.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
			}
			h.mutex.Unlock()

			// 通知所有客户端有用户离开
			msg := Message{
				Type:   "user_left",
				UserID: client.UserID,
				Time:   time.Now().Unix(),
			}
			msgBytes, _ := json.Marshal(msg)
			h.Broadcast <- msgBytes

		case message := <-h.Broadcast:
			h.mutex.RLock()
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// WebSocket 处理函数
func handleWebSocket(c *gin.Context) {
	// 从 Gin 的 Context 中获取原生 http.ResponseWriter 和 Request
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket 升级失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "WebSocket 连接失败"})
		return
	}

	userID := c.Query("user_id")
	if userID == "" {
		userID = "anonymous_" + time.Now().Format("20060102150405")
	}

	client := &Client{
		ID:     c.ClientIP() + "_" + time.Now().Format("150405"),
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userID,
	}

	hub.Register <- client

	// 启动读写协程
	go client.writePump()
	go client.readPump()

	log.Printf("客户端连接: %s (用户: %s)", client.ID, client.UserID)
}

func (c *Client) readPump() {
	defer func() {
		hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("读取错误: %v", err)
			}
			break
		}

		// 处理收到的消息
		var msgData Message
		if err := json.Unmarshal(message, &msgData); err == nil {
			msgData.UserID = c.UserID
			msgData.Time = time.Now().Unix()

			// 设置消息类型（如果没有）
			if msgData.Type == "" {
				msgData.Type = "message"
			}

			// 广播消息
			msgBytes, _ := json.Marshal(msgData)
			hub.Broadcast <- msgBytes
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// 通道关闭
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 写入队列中的其他消息
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			// 发送心跳
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// HTTP API 路由
func setupRouter() *gin.Engine {
	r := gin.Default()

	// 添加 CORS 中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// WebSocket 路由
	r.GET("/ws", handleWebSocket)

	// RESTful API 路由
	api := r.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":    "healthy",
				"timestamp": time.Now().Unix(),
				"clients":   len(hub.Clients),
			})
		})

		api.GET("/clients", func(c *gin.Context) {
			hub.mutex.RLock()
			clientCount := len(hub.Clients)
			hub.mutex.RUnlock()

			c.JSON(200, gin.H{
				"total_clients": clientCount,
			})
		})

		api.POST("/broadcast", func(c *gin.Context) {
			var msg struct {
				Message string `json:"message" binding:"required"`
			}

			if err := c.ShouldBindJSON(&msg); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			broadcastMsg := Message{
				Type:    "admin",
				Content: msg.Message,
				Time:    time.Now().Unix(),
				UserID:  "system",
			}

			msgBytes, _ := json.Marshal(broadcastMsg)
			hub.Broadcast <- msgBytes

			c.JSON(200, gin.H{"status": "message sent"})
		})
	}

	// 静态文件服务
	r.Static("/static", "./public")
	r.GET("/", func(c *gin.Context) {
		c.File("./public/index.html")
	})

	return r
}

func main() {
	// 启动 WebSocket hub
	go hub.Run()

	// 设置 Gin 路由
	r := setupRouter()

	// 启动服务器
	log.Println("服务器启动在 :8080")
	log.Println("WebSocket 端点: ws://localhost:8080/ws")
	log.Println("HTTP API 端点: http://localhost:8080/api")

	if err := r.Run(":8080"); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}
