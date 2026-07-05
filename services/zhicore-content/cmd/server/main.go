package main

import (
	"fmt"
	"log"

	contentruntime "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/runtime"
)

func main() {
	if err := buildModule(); err != nil {
		log.Fatal(err)
	}
}

func buildModule() error {
	// 当前切片只建立进程根和 runtime 边界。真实 PostgreSQL、MongoDB、
	// User/File clients、outbox dispatcher 和配置加载落地前必须 fail fast，
	// 避免把未装配服务伪装成生产可运行实例。
	if _, err := contentruntime.Build(contentruntime.Deps{}); err != nil {
		return fmt.Errorf("build content runtime module: %w", err)
	}
	return nil
}
