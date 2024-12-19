package main

import (
	"context"
	"fmt"
	"github.com/eatmoreapple/openwechat"
	openai "github.com/sashabaranov/go-openai"
	"strings"
	"time"
)

// TIP To run your code, right-click the code and select <b>Run</b>. Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.
func main() {
	bot := openwechat.DefaultBot(openwechat.Desktop) // 桌面模式
	messages := make([]openai.ChatCompletionMessage, 0)
	config := openai.DefaultConfig("your chatGPT api key here")
	config.BaseURL = "https://api.chatanywhere.tech"
	client := openai.NewClientWithConfig(config)
	// 注册消息处理函数
	bot.MessageHandler = func(msg *openwechat.Message) {
		if msg.IsText() {
			// 打印 msg.Content
			fmt.Println(msg.Content)
		}
		if msg.IsAt() {
			// check msg.Content starts with "@akka "
			if !strings.Contains(msg.Content, "@akka") {
				return
			}
			now := time.Now().Hour()
			if now <= 7 || now >= 20 {
				msg.ReplyText(openwechat.Emoji.Shhh + "~ 晚安" + openwechat.Emoji.Sleep + openwechat.Emoji.Sleep)
				return
			}
			// 截取 "@akka "后面的部分
			question := strings.ReplaceAll(msg.Content, "@akka", "")
			fmt.Printf(question)

			if question == "skadoosh" {
				// 清空messages
				messages = make([]openai.ChatCompletionMessage, 0)
			}

			// 加入上下文
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: question,
			})
			answer := ""
			resp, err := client.CreateChatCompletion(
				context.Background(),
				openai.ChatCompletionRequest{
					Model:    openai.GPT3Dot5Turbo,
					Messages: messages,
				},
			)
			if err != nil {
				fmt.Printf("ChatCompletion error: %v\n", err)
				messages = make([]openai.ChatCompletionMessage, 0)
				messages = append(messages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: question,
				})
				// 重试
				res, err := client.CreateChatCompletion(
					context.Background(),
					openai.ChatCompletionRequest{
						Model:    openai.GPT3Dot5Turbo,
						Messages: messages,
					},
				)
				if err != nil {
					fmt.Printf("ChatCompletion error: %v\n", err)
					return
				} else {
					answer = res.Choices[0].Message.Content
				}
			} else {
				answer = resp.Choices[0].Message.Content
			}
			// 回答加入上下文
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: answer,
			})
			msg.ReplyText(answer)

		}
		if msg.IsTickledMe() {
			msg.ReplyText("再拍报警了")
		}
	}
	// 注册登陆二维码回调
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl

	// 登陆
	if err := bot.Login(); err != nil {
		fmt.Println(err)
		return
	}

	// 获取登陆的用户
	self, err := bot.GetCurrentUser()
	if err != nil {
		fmt.Println(err)
		return
	}

	// 获取所有的好友
	friends, err := self.Friends()
	fmt.Println(friends, err)

	// 获取所有的群组
	groups, err := self.Groups()
	fmt.Println(groups, err)

	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	bot.Block()
}

//TIP See GoLand help at <a href="https://www.jetbrains.com/help/go/">jetbrains.com/help/go/</a>.
// Also, you can try interactive lessons for GoLand by selecting 'Help | Learn IDE Features' from the main menu.
