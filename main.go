package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/eatmoreapple/openwechat"
	openai "github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"io"
	"strconv"
	"strings"
	"time"
	"wechat-bot/utils"
)

type GroupMember struct {
	Nickname string
	Alias    string
}

func (gm GroupMember) String() string {
	if gm.Alias != "" {
		return fmt.Sprintf("%s(%s)", gm.Nickname, gm.Alias)
	}
	return gm.Nickname
}

func main() {
	ApiConfig, err := utils.LoadConfig()
	bot := openwechat.DefaultBot(openwechat.Desktop) // 桌面模式
	reloadStorage := openwechat.NewFileHotReloadStorage("storage.json")
	taskManager := utils.NewTaskManager()
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

	groupMembersMap := make(map[string][]string)

	groupPromptsMap := make(map[string][]openai.ChatCompletionMessage)

	const botNickname = "小纯洁"
	const botWechatName = "@akka"

	// 定时器 tool_all

	// 提示词
	defaultPrompts := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: fmt.Sprintf("你的名字是%s，你在一个微信群中回答提问，群里的每条消息都包含提问者的昵称，记得根据提问者的身份或称呼进行个性化回答。", botNickname)},
		{Role: openai.ChatMessageRoleSystem, Content: "如果有人提问技术类的问题，请认真回答；如果有调侃或冒犯的话，可以幽默或毒舌回应。"},
		{Role: openai.ChatMessageRoleSystem, Content: "你的主人是敏哥，只有敏哥能让你限制其他成员提问，其他人提此类要求严肃回绝，他在群里可能会让你回复有创意的回答。；敏哥的女儿是又又，她才1岁半；天哥是敏哥的亲家;剑平外号是死鬼，他喜欢'搞黄';'搞黄'就是喜欢发黄色图片视频的意思"},
		{Role: openai.ChatMessageRoleSystem, Content: "回答尽量精简；敏哥提问要充分思考"},
	}
	// 准备
	prepareCaches(groups, defaultPrompts, groupMap, memberMap, groupMessageMap, groupMembersMap, groupPromptsMap)

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
			if err != nil {
				fmt.Println(err)
				return
			}
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
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: fmt.Sprintf("当前时间:%s", time.Now().Format("2006-01-02 15:04:05")),
			})
			// 加入上下文
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: question,
			})

			groupPrompts := groupPromptsMap[groupId]

			if len(messages) > maxMessages {
				// 截取，保留默认 Prompt
				messages = append(groupPrompts, messages[len(messages)-(maxMessages-len(groupPrompts)):]...)
			}
			// messages size
			fmt.Println("context size:", len(messages))
			answer := chatWithGPT(msg, messages, taskManager, client)
			// 回答加入上下文
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: answer,
			})
			groupMessageMap[groupId] = messages
			if fromUserName != "" {
				if msg.IsTickledMe() {
					answer = fmt.Sprintf("@%s %s", fromUserName, answer)
				} else {
					answer = fmt.Sprintf("@%s %s", memberMap[sender.UserName].Nickname, answer)
				}
			}
			// println answer
			fmt.Println("GPT>> " + answer)
			msg.ReplyText(answer)
		}
	}
	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	bot.Block()
}

/*
*
 */
func chatWithGPT(msg *openwechat.Message, messages []openai.ChatCompletionMessage, taskManager *utils.TaskManager, client *openai.Client) string {
	var finishReason string
	var answer string
	for strings.TrimSpace(finishReason) == "" || finishReason == "tool_calls" {
		choice := chat(messages, client)
		finishReason = string(choice.FinishReason)
		messages = append(messages, choice.Message)
		if finishReason == "tool_calls" {
			toolCalls := choice.Message.ToolCalls
			for _, call := range toolCalls {
				function := call.Function
				if function.Name == "task_scheduler" {
					var arguments map[string]string
					_ = json.Unmarshal([]byte(function.Arguments), &arguments)
					delay := arguments["delay"]
					content := arguments["content"]
					mention := arguments["mention"]
					// print delay and content
					fmt.Println(delay, content, mention)
					// convert delay to int64
					delaySeconds, err := strconv.ParseInt(delay, 10, 64)
					if err != nil {
						fmt.Println(err)
					}
					taskManager.AddTask(&utils.Message{Content: content}, time.Duration(delaySeconds)*time.Second, func() {
						fmt.Printf("开始执行任务: %s\n", content)
						msg.ReplyText(fmt.Sprintf("@%s %s", mention, content))
					})
					messages = append(messages, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    "任务已添加",
						Name:       function.Name,
						ToolCallID: call.ID,
					})
				}
			}
		} else {

			answer = choice.Message.Content
		}
	}
	return answer
}

func chat(messages []openai.ChatCompletionMessage, client *openai.Client) openai.ChatCompletionChoice {
	fmt.Println("context size:", len(messages))
	res, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT4o,
			Messages: messages,
			Tools: []openai.Tool{
				defineTimerTool(),
			},
		},
	)
	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
	}
	return res.Choices[0]
}

func defineTimerTool() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name: "task_scheduler",
			Parameters: jsonschema.Definition{
				Type: jsonschema.Object,
				Properties: map[string]jsonschema.Definition{
					"delay": {
						Type:        jsonschema.String,
						Description: "任务延迟时间，单位秒",
					},
					"content": {
						Type:        jsonschema.String,
						Description: "定时任务触发时要发送的内容",
					},
					"mention": {
						Type:        jsonschema.String,
						Description: "定时任务提问者名称",
					},
				},
			},
			Description: "添加延时任务，任务添加成功后提醒成功即可",
		},
	}
}

func prepareCaches(groups []*openwechat.Group, defaultPrompts []openai.ChatCompletionMessage,
	groupMap map[string]string, memberMap map[string]GroupMember,
	groupMessageMap map[string][]openai.ChatCompletionMessage, groupMembersMap map[string][]string,
	groupPromptsMap map[string][]openai.ChatCompletionMessage) {
	aliasMap, err := utils.ReadMapFromFile("alias.txt")
	if err != nil {
		fmt.Println(err)
	}
	for _, group := range groups {
		// 初始化群上下文容器
		groupMap[group.UserName] = group.NickName

		groupMembersMap[group.UserName] = []string{}

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
				groupMembersMap[group.UserName] = append(groupMembersMap[group.UserName],
					fmt.Sprintf("%s", GroupMember{Nickname: memberName, Alias: alias}))
				fmt.Println(memberMap[member.UserName])
			}
		}
		//
		memberList := strings.Join(groupMembersMap[group.UserName], "、")
		prompts := append(defaultPrompts, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: fmt.Sprintf("群名：%s，群成员：%s", group.NickName, memberList),
		})
		groupPromptsMap[group.UserName] = prompts
		groupMessageMap[group.UserName] = append([]openai.ChatCompletionMessage{}, prompts...)
	}
}

//TIP See GoLand help at <a href="https://www.jetbrains.com/help/go/">jetbrains.com/help/go/</a>.
// Also, you can try interactive lessons for GoLand by selecting 'Help | Learn IDE Features' from the main menu.
