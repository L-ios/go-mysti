package controllers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/go-redis/redis/v8"
)

func client_test() {
	// 加载 CA 证书
	caCert, err := os.ReadFile("./redis-ssl/ca.crt")
	if err != nil {
		fmt.Printf("读取 CA 证书失败: %v\n", err)
		return
	}

	// 创建证书池
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		fmt.Println("添加 CA 证书失败")
		return
	}

	// 加载客户端证书和密钥（如果 Redis 要求客户端证书）
	cert, err := tls.LoadX509KeyPair("./redis-ssl/redis.crt", "./redis-ssl/redis.key")
	if err != nil {
		fmt.Printf("加载客户端证书/密钥失败: %v\n", err)
		return
	}

	// 配置 TLS
	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
	}

	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr:      "localhost:6379",
		Password:  "", // 如果设置了 requirepass，填入密码
		DB:        0,
		TLSConfig: tlsConfig,
	})

	// 创建上下文
	ctx := context.Background()

	// 测试连接
	pong, err := client.Ping(ctx).Result()
	if err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}
	fmt.Println("连接成功:", pong)

	// 示例：设置和获取键值
	err = client.Set(ctx, "key", "value", 0).Err()
	if err != nil {
		fmt.Printf("设置键失败: %v\n", err)
		return
	}

	val, err := client.Get(ctx, "key").Result()
	if err != nil {
		fmt.Printf("获取键失败: %v\n", err)
		return
	}
	fmt.Println("键值:", val)

	// 关闭客户端
	client.Close()
}
