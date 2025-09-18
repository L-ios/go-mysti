package controllers

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

const (
	KUBERNETES_API_SERVER = "https://kubernetes.default.svc"
	SERVICE_ACCOUNT_PATH  = "/var/run/secrets/kubernetes.io/serviceaccount"
	KUBERNETES_CA_CERT    = SERVICE_ACCOUNT_PATH + "/ca.crt"
	KUBERNETES_TOKEN      = SERVICE_ACCOUNT_PATH + "/token"
	KUBERNETES_NAMESPACE  = SERVICE_ACCOUNT_PATH + "/namespace"
)

type KubeController struct {
}

func RegisterKubeRoutes(router *gin.RouterGroup) KubeController {
	ctl := KubeController{}
	router.GET("/*kubernetesPath", ctl.proxy)

	return ctl
}

func (ctrl KubeController) proxy(ctx *gin.Context) {
	targetUri := ctx.Param("kubernetesPath")

	// 1. 读取 ServiceAccount token
	token, err := os.ReadFile(KUBERNETES_TOKEN)
	if err != nil {
		panic(err)
	}

	// 2. 读取 CA 证书
	caCert, err := os.ReadFile(KUBERNETES_CA_CERT)
	if err != nil {
		panic(err)
	}

	// 4. 构造 TLS 客户端
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	// 5. 构造请求
	url := fmt.Sprintf("%s%s", KUBERNETES_API_SERVER, targetUri)

	req, err := http.NewRequest(ctx.Request.Method, url, ctx.Request.Body)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Bearer "+string(token))

	// 6. 发送请求
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	ctx.Header("Content-Type", resp.Header.Get("Content-Type"))
	ctx.Writer.Write(body)
}
