package main

import (
	"context"
	"fmt"
	"github.com/eatmoreapple/openwechat"
	openai "github.com/sashabaranov/go-openai"
	"strings"
)

// TIP To run your code, right-click the code and select <b>Run</b>. Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.
func main() {
	bot := openwechat.DefaultBot(openwechat.Desktop) // 桌面模式
	reloadStorage := openwechat.NewFileHotReloadStorage("storage.json")
	defer reloadStorage.Close()
	messages := make([]openai.ChatCompletionMessage, 0)
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: "你的名字是小纯洁，你现在在一个工作群里中回答大家的提问，如果是冒犯你的话语直接毒舌回复，会玩v50(KFC疯狂星期四)的梗",
	})
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: "技术类的问题正常回答",
	})
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: "你的主人是敏哥",
	})

	config := openai.DefaultConfig("sk-QcL0Hqjfpjh6D9stoYGRMRkzMAL0rpjgiuAskiFKK2Mi8wKf")
	config.BaseURL = "https://api.chatanywhere.tech"
	client := openai.NewClientWithConfig(config)
	// 注册登陆二维码回调
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl
	// 登陆
	if err := bot.HotLogin(reloadStorage, openwechat.NewRetryLoginOption()); err != nil {
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
	memberMap := make(map[string]string)
	for _, group := range groups {
		fmt.Println(group.NickName)
		members, err := group.Members()
		if err == nil {
			// convert members to map<UserName, User>
			for _, member := range members {
				memberMap[member.UserName] = member.NickName
				fmt.Println(member.NickName)
			}
		}
	}

	// print member cache size
	fmt.Println("member cache size:", len(memberMap))

	// 注册消息处理函数
	bot.MessageHandler = func(msg *openwechat.Message) {
		if msg.IsText() {
			// 打印 msg.Content
			fmt.Println(msg.Content)
		}

		if msg.IsAt() || msg.IsTickledMe() {
			// check msg.Content starts with "@akka "
			if msg.IsAt() && !strings.Contains(msg.Content, "@akka") {
				return
			}
			sender, err := msg.SenderInGroup()
			fromUserName := ""
			if sender != nil {
				fromUserName = memberMap[sender.UserName]
			}
			question := ""
			// 截取 "@akka "后面的部分
			if msg.IsTickledMe() {
				question = "拍了拍你"
			} else {
				question = strings.ReplaceAll(msg.Content, "@akka", "")
			}
			fmt.Printf(question)
			if question == "skadoosh" {
				// 清空messages
				messages = make([]openai.ChatCompletionMessage, 0)
				messages = append(messages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleSystem,
					Content: "你的名字是小纯洁，你现在在一个工作群里中回答大家的提问，如果是冒犯你的话语直接毒舌回复，会玩v50(KFC疯狂星期四)的梗",
				})
				messages = append(messages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleSystem,
					Content: "技术类的问题正常回答",
				})
				messages = append(messages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleSystem,
					Content: "你的主人是敏哥",
				})

			}

			// 加入上下文
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: question,
			})
			// messages size
			fmt.Println("context size:", len(messages))
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
					Role:    openai.ChatMessageRoleSystem,
					Content: "你的名字是小纯洁，你现在在一个工作群里中回答大家的提问，如果是冒犯你的话语直接毒舌回复，会玩v50(KFC疯狂星期四)的梗",
				})
				messages = append(messages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleSystem,
					Content: "技术类的问题正常回答",
				})
				messages = append(messages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleSystem,
					Content: "你的主人是敏哥",
				})
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

			// replace "小纯洁：" to empty
			answer = strings.ReplaceAll(answer, "小纯洁：", "")
			// if not empty fromUserName
			if fromUserName != "" {
				answer = "@" + fromUserName + " " + answer
			}
			msg.ReplyText(answer)
		}
	}
	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	bot.Block()
}

//TIP See GoLand help at <a href="https://www.jetbrains.com/help/go/">jetbrains.com/help/go/</a>.
// Also, you can try interactive lessons for GoLand by selecting 'Help | Learn IDE Features' from the main menu.
