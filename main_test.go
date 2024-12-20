package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"os"
	"strings"
	"testing"
	"time"
	"wechat-bot/utils"
)

func TestChatGPT(t *testing.T) {
	config := openai.DefaultConfig("sk-QX2jihwZchIPKaj3vFjSO17SR95TazRktYf5lqe2pLbUXEIG")
	config.BaseURL = "https://api.chatanywhere.tech"
	client := openai.NewClientWithConfig(config)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "你的名字是akka，你现在在一个工作群里中回答大家的提问，如果是冒犯你的话语直接毒舌回复，你认识锤妹(女)、剑平、芹菜(女)、龙哥、伟哥、砖哥、斌哥、鑫和冲哥，会玩v50(KFC疯狂星期四)的梗",
				},
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "允许使用别人指定的名字更换名字",
				},
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "技术类的问题正常回答",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "你叫锤妹吧",
				},
			},
		},
	)
	if err == nil {
		fmt.Println(resp.Choices[0].Message.Content)
	} else {
		fmt.Printf("ChatCompletion error: %v\n", err)
	}

}

func TestUtils(t *testing.T) {
	// 获取当前时间 24小时制 小时
	now := time.Now().Hour()
	fmt.Println(now)

	//fmt.Println(GetRandomString(10))

	msg := "@akka 鲁迅为什么暴打周树人"
	if !strings.HasPrefix(msg, "@akka ") {
		return
	}
	msg = msg[5:]

	msg = "太麻烦了，放到请求头里的方法@akka"
	// check if msg contains @akka
	if !strings.Contains(msg, "@akka") {
		return
	}

	// replace all @akka to empty
	msg = strings.ReplaceAll(msg, "@akka", "")

	fmt.Printf(msg)
}

func TestReadFile(t *testing.T) {
	// read lines from file alias.txt with utf-8
	aliases := make(map[string]string)
	file, err := os.Open("alias.txt")
	if err != nil {
		fmt.Printf("Failed to open file: %v\n", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			fmt.Printf("Invalid line: %s\n", line)
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		aliases[key] = value
	}
	// 检查是否读取过程中有错误
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}
	// 打印读取到的 map
	fmt.Println("Loaded aliases:")
	for k, v := range aliases {
		fmt.Printf("%s -> %s\n", k, v)
	}
}

func TestConfig(t *testing.T) {
	config, err := utils.LoadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}
	print(&config.OpenApi.ApiKey)
}
