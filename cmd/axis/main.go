package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Waasaabii/AXIS/internal/axis"
)

func main() {
	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "serve":
		if err := serve(); err != nil {
			log.Fatal(err)
		}
	case "reset-setup":
		targetConfigPath := configPath()
		if len(os.Args) > 2 && os.Args[2] != "" {
			targetConfigPath = os.Args[2]
		}
		if err := resetSetup(targetConfigPath); err != nil {
			log.Fatal(err)
		}
	case "hash-password":
		if len(os.Args) < 3 {
			log.Fatal("请提供需要哈希的密码")
		}
		hash, err := axis.CreatePasswordHash(os.Args[2])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(hash)
	case "preflight":
		if err := printPreflight(); err != nil {
			log.Fatal(err)
		}
	case "openapi":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(axis.BuildOpenAPISpec()); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("未知命令: %s", command)
	}
}

func configPath() string {
	if value := os.Getenv("PROXYRELAY_CONFIG"); value != "" {
		resolvedPath, err := axis.EnsureConfigPath(value)
		if err != nil {
			log.Fatal(err)
		}
		return resolvedPath
	}
	resolvedPath, err := axis.EnsureConfigPath("")
	if err != nil {
		log.Fatal(err)
	}
	return resolvedPath
}

func serve() error {
	service, err := axis.NewService(configPath())
	if err != nil {
		return err
	}
	defer service.Close()

	server := axis.NewServer(service)
	address := fmt.Sprintf("%s:%d", service.Config().Server.Host, service.Config().Server.Port)
	log.Printf("[axis] 控制面已启动: http://%s\n", address)
	return http.ListenAndServe(address, server)
}

func printPreflight() error {
	service, err := axis.NewService(configPath())
	if err != nil {
		return err
	}
	defer service.Close()
	snapshot := service.GetRuntimePreflight()
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(snapshot)
}

func resetSetup(targetConfigPath string) error {
	result, err := axis.ResetSetup(targetConfigPath)
	if err != nil {
		return err
	}

	fmt.Printf("[axis] 已重置 Setup 状态\n")
	fmt.Printf("[axis] 配置文件: %s\n", result.ConfigPath)
	fmt.Printf("[axis] 已清理运行目录: %s\n", result.RuntimeDir)
	fmt.Printf("[axis] 下次启动后会重新进入 Setup 流程\n")
	return nil
}
