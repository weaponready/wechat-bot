package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestChatGPT(t *testing.T) {
	/*config := openai.DefaultConfig("sk-QcL0Hqjfpjh6D9stoYGRMRkzMAL0rpjgiuAskiFKK2Mi8wKf")
	config.BaseURL = "https://api.chatanywhere.tech"
	client := openai.NewClientWithConfig(config)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "hi",
				},
			},
		},
	)
	if err == nil {
		fmt.Println(resp.Choices[0].Message.Content)
	}*/
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
