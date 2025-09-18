package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var kubeconfig string

func init() {
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
}

var podsCmd = &cobra.Command{
	Use:   "pods",
	Short: "List pods in current namespace",
	Run: func(cmd *cobra.Command, args []string) {
		// 使用kubeconfig文件创建配置
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			fmt.Printf("Error building kubeconfig: %v\n", err)
			os.Exit(1)
		}

		// 创建clientset
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			fmt.Printf("Error creating clientset: %v\n", err)
			os.Exit(1)
		}

		// 获取pods列表
		pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Printf("Error listing pods: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Found %d pods:\n", len(pods.Items))
		for _, pod := range pods.Items {
			fmt.Printf("- %s (namespace: %s)\n", pod.Name, pod.Namespace)
		}
	},
}
