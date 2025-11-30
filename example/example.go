package main

import (
	"fmt"
	"github.com/gaozhiheng/myconfig"
	"log"
)

// go build -ldflags '-X github.com/gaozhiheng/myconfig.keyFilePassword=Gao@2025' -o example example.go
func main() {
	// 初始化配置（不再需要传递解密密码）
	err := myconfig.Init("config.json", "myconfigkey.json")
	if err != nil {
		log.Fatal(err)
	}

	// 正常使用配置
	myconfig.SetConfig("CLIENT_ID", "12345")
	myconfig.SetConfig("Port", "8080")
	clientID := myconfig.GetString("CLIENT_ID")
	port := myconfig.GetInt("PORT")

	fmt.Println(clientID)
	fmt.Println(port)
	// 重置密码
	//err = myconfig.SetPass("new_config_password")
	//if err != nil {
	//	log.Fatal(err)
	//}
}
