package main

import (
	"context"
	"fmt"
	"github.com/eatmoreapple/openwechat"
	openai "github.com/sashabaranov/go-openai"
	"io"
	"strings"
	"wechat-bot/utils"
)

type GroupMember struct {
	Nickname string
	Alias    string
}

func main() {
	ApiConfig, err := utils.LoadConfig()
	bot := openwechat.DefaultBot(openwechat.Desktop) // 桌面模式
	reloadStorage := openwechat.NewFileHotReloadStorage("storage.json")
	defer func(reloadStorage io.ReadWriteCloser) {
		err := reloadStorage.Close()
		if err != nil {
			// print error
			fmt.Println(err)
		}
	}(reloadStorage)
	// print apikey and baseurl
	fmt.Println(ApiConfig.OpenApi.ApiKey, ApiConfig.OpenApi.BaseUrl)
	config := openai.DefaultConfig(ApiConfig.OpenApi.ApiKey)
	config.BaseURL = ApiConfig.OpenApi.BaseUrl
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
	// 人员名称映射
	memberMap := make(map[string]GroupMember)
	// 群组名称映射
	groupMap := make(map[string]string)
	// 群消息上下文
	groupMessageMap := make(map[string][]openai.ChatCompletionMessage)

	const botNickname = "小纯洁"
	const botWechatName = "@akka"

	// 提示词
	defaultPrompts := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: fmt.Sprintf("你的名字是%s，你在一个微信群中回答提问，群里的每条消息都包含提问者的昵称，记得根据提问者的身份或称呼进行个性化回答。", botNickname)},
		{Role: openai.ChatMessageRoleSystem, Content: "如果有人提问技术类的问题，请认真回答；如果有调侃或冒犯的话，可以幽默或毒舌回应。"},
		{Role: openai.ChatMessageRoleSystem, Content: "你的主人是敏哥，他在群里可能会让你回复有创意的回答。；敏哥的女儿是又又，她才1岁半；天哥是敏哥的亲家"},
	}
	// 准备
	prepareCaches(groups, defaultPrompts, groupMap, memberMap, groupMessageMap)

	// print member cache size
	fmt.Println("member cache size:", len(memberMap))
	const maxMessages = 20
	// 注册消息处理函数
	bot.MessageHandler = func(msg *openwechat.Message) {
		if msg.IsText() {
			// 打印 msg.Content
			fmt.Println(msg.Content)
		}

		if (msg.IsAt() || msg.IsTickledMe()) && msg.IsComeFromGroup() {
			// check msg.Content starts with "@akka "
			if msg.IsAt() && !strings.Contains(msg.Content, botWechatName) {
				return
			}

			groupId := msg.FromUserName
			messages := groupMessageMap[groupId]
			sender, err := msg.SenderInGroup()
			fromUserName := ""
			if sender != nil {
				fromUserName = memberMap[sender.UserName].Alias
			} else if msg.IsTickledMe() {
				fromUserName = strings.ReplaceAll(msg.Content, "拍了拍我", "")
				fromUserName = strings.ReplaceAll(fromUserName, "\"", "")
			}
			question := ""
			// 截取 "@akka "后面的部分
			if msg.IsTickledMe() {
				question = fromUserName + ": 拍了拍你"
			} else {
				question = strings.ReplaceAll(msg.Content, botWechatName, "")
				if fromUserName != "" {
					question = fmt.Sprintf("%s提问：%s", fromUserName, question)
				}
			}
			fmt.Printf(fromUserName + ":" + question)
			// 加入上下文
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: question,
			})
			if len(messages) > maxMessages {
				// 截取，保留默认 Prompt
				messages = append(defaultPrompts, messages[len(messages)-(maxMessages-len(defaultPrompts)):]...)
			}
			// messages size
			fmt.Println("context size:", len(messages))
			answer := ""
			resp, err := client.CreateChatCompletion(
				context.Background(),
				openai.ChatCompletionRequest{
					Model:    openai.GPT4o,
					Messages: messages,
				},
			)
			if err != nil {
				fmt.Printf("ChatCompletion error: %v\n", err)
				messages = append(defaultPrompts, messages[len(messages)-(maxMessages-len(defaultPrompts)):]...)
				// 重试
				res, err := client.CreateChatCompletion(
					context.Background(),
					openai.ChatCompletionRequest{
						Model:    openai.GPT4o,
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
			groupMessageMap[groupId] = messages
			if fromUserName != "" {
				answer = fmt.Sprintf("@%s %s", memberMap[sender.UserName].Nickname, answer)
			}
			// println answer
			fmt.Println("GPT>> " + answer)
			msg.ReplyText(answer)
		}
	}
	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	bot.Block()
}

func prepareCaches(groups []*openwechat.Group, defaultPrompts []openai.ChatCompletionMessage,
	groupMap map[string]string, memberMap map[string]GroupMember, groupMessageMap map[string][]openai.ChatCompletionMessage) {
	aliasMap, err := utils.ReadMapFromFile("alias.txt")
	if err != nil {
		fmt.Println(err)
	}
	for _, group := range groups {
		// 初始化群上下文容器
		groupMap[group.UserName] = group.NickName

		groupMessageMap[group.UserName] = append([]openai.ChatCompletionMessage{}, defaultPrompts...)
		fmt.Println(group.NickName)
		members, err := group.Members()
		if err == nil {
			// convert members to map<UserName, User>
			for _, member := range members {
				memberName := ""
				// member.alias or member.nickName
				if member.Alias != "" {
					memberName = member.Alias
				} else {
					memberName = member.NickName
				}
				alias := aliasMap[memberName]
				if alias != "" {
					memberMap[member.UserName] = GroupMember{Nickname: memberName, Alias: alias}
				} else {
					memberMap[member.UserName] = GroupMember{Nickname: memberName, Alias: memberName}
				}
				fmt.Println(memberMap[member.UserName])
			}
		}
	}
}

//TIP See GoLand help at <a href="https://www.jetbrains.com/help/go/">jetbrains.com/help/go/</a>.
// Also, you can try interactive lessons for GoLand by selecting 'Help | Learn IDE Features' from the main menu.
