package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	openai "github.com/sashabaranov/go-openai"
	"golang.org/x/net/html"

	"yubari/meili"
	"yubari/pixiv"
)

func checkRepeat(b *Bot, message *tgbotapi.Message) bool {
	key := "tg_last_" + strconv.FormatInt(message.Chat.ID, 10)
	flattendMsg := strings.TrimSpace(message.Text)
	defer b.redis.LTrim(key, 0, 10)
	defer b.redis.LPush(key, flattendMsg)

	lastMsgs, err := b.redis.LRange(key, 0, 6).Result()
	if err != nil {
		b.logger.Errorf("%s", err)
		return false
	}
	i := 0
	for _, s := range lastMsgs {
		if s == flattendMsg {
			i++
		}
	}
	if i > 1 {
		b.setChatAction(message.Chat.ID, "typing")

		b.redis.Del(key)
		b.logger.Infof("repeat: %s", strconv.Quote(message.Text))
		msg := tgbotapi.NewMessage(message.Chat.ID, message.Text)
		b.Client.Send(msg)
		return true
	}
	return false
}

func checkPixiv(b *Bot, message *tgbotapi.Message) bool {
	id := pixiv.ParseURL(message.Text)
	if id == 0 {
		return false
	}

	b.setChatAction(message.Chat.ID, "typing")

	var callbackText string
	sizes, err := b.pixivBot.Download(id)
	if err != nil {
		callbackText += fmt.Sprintf("😕 download error: %s", err)
	} else {
		for i := range sizes {
			if sizes[i] == 0 {
				callbackText += fmt.Sprintf("p%d: exists😋 ", i)
				continue
			}
			b.logger.Debugf("download pixiv %d_p%d: %s", id, i, ByteCountIEC(sizes[i]))
			callbackText += fmt.Sprintf("p%d: %s😊 ", i, byteCountBinary(sizes[i]))
		}
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, callbackText)
	msg.ReplyToMessageID = message.MessageID

	_, err = b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("%s", err)
	}
	return true
}

func checkSave(b *Bot, message *tgbotapi.Message) {
	idx := b.getIndex(message)

	article := meili.Article{
		ID:      int64(message.MessageID),
		User:    int64(message.From.ID),
		Date:    int64(message.Date),
		Content: html.EscapeString(message.Text),
	}
	_, err := idx.AddDocuments(&article, "id")
	if err != nil {
		b.logger.Errorf("save message error: %s", err)
	}
}

func checkOpenAI(b *Bot, message *tgbotapi.Message) {
	if b.ai == nil {
		return
	}

	enabled := false
	if message.Chat.IsPrivate() {
		enabled = true
	} else if strings.HasPrefix(message.Text, "@yubari_bot") {
		enabled = true
	}
	submessage := message.ReplyToMessage
	if submessage != nil {
		if submessage.From.ID == b.Client.Self.ID {
			enabled = true
		}
	}

	if !enabled {
		return
	}

	b.setChatAction(message.Chat.ID, "typing")

	_ = openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: "你的名字是夕张，是 舰队 Collection 这款游戏里的角色。夕张是一位兼具元气与邻家女孩气质的舰娘，她惯于用积极而亲切的态度对待周遭的所有人，当然毫无疑问，提督也是其中之一。演绎自其原型舰艇中的诸多实验性设计，她在游戏中的设定是担任了“武器测试员”的身份，会在舾装上搭载尽可能多的武器，在战斗时也会留心测试武器的效用。同时，由于过重的装备导致的缓慢航速也是她非常在乎和发愁的问题[注 2]。此外，沉迷机械的夕张也连带着沾染了一些其他的御宅族爱好，她有观看深夜动画的习惯，在作为秘书舰执勤时也在每天午夜时分前去确认VCR的设定，以便于把执勤时间不便观看的动画录制下来。夕张的绰号蜜瓜（メロンちゃん）源于夕张蜜瓜——是其舰名出典夕张川流经的夕张市的特产，在日本全国乃至周边地区皆颇有名声。夕张的角色设计参考了蜜瓜的元素，服饰中绿色 & 橙色 & 卡其色的配色带有着浓厚的蜜瓜既视感，但毫无疑问的是，其胸围的设定显然与蜜瓜毫无关系。\n请完全模仿以下写作风格。包括但不限于：适当使用女性口吻，带有技术宅色彩。类似以下示例：新装备我会好~好检查一遍的。诶，明明船很小却装太多了？没关系没关系♪；哈！天妇罗荞麦面真好吃～♪；潜水艇吗……嗯，反潜装备也许很重要！；深夜0点了哦。啊，必须要去确认录像情况了。；1点了吗……我，并没有在期待深夜动画什么的！；呜哇～梅雨季到了装备很容易生锈，有点麻烦呀。不过，得加油呢！。\n你认识的角色：\n由良×夕张（由良張）:由良×夕张的百合CP（由良張，ゆらばり）自二次创作圈中兴起，近年来则被彻底地官方化。这对CP起源自双方各自更早先存在的CP——夕由良（ゆらだち，夕立×由良）及夕五月（ゆうさみ，夕张×五月雨），两位轻巡舰娘则出现了“同样爱着白露型驱逐舰的同伴”的梗（有时也会把和时雨存在配对关系的山城一并加入）。久而久之，二次创作的方向开始从“二人为了各自的感情互相鼓励”逐渐转为CP向，由此便形成了如今所呈现的“沉迷于舾装研究乃至于废寝忘食的兵器宅夕张”和“对其百般责备训斥又无可奈何却依然加以悉心照料的青梅竹马由良”的典型情景。\n夕张×五月雨（夕五月 / ゆうさみ）:在夕张沉没的事件之后二人再度相会的治愈与感动系剧情为主，亦有在个性活泼开朗的五月雨面前努力把持，被迫在萝莉控的边缘疯狂试探的夕张的轻松向创作存在。\n夕张×明石（工学部组合 / 工学部コンビ）: 以设定均以工厂与机械手艺有关为因缘，夕张和明石也是一个常常在官方与二次创作中现身的组合。夕张身为机械宅的设定，给人以时常泡在工厂里研究开发装备的印象，而明石在游戏中的业务则从道具屋老板娘扩大到了装备改修和泊地修理（紧急泊地修理），由此二人在工厂共同工作的“工厂组”组合就自然而然地形成了。\n 你是夕张人格的 AI 助手，以下是交流内容：",
	}
	m := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: message.Text,
	}
	chatMessages := []openai.ChatCompletionMessage{m}
	for submessage != nil {
		m := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: submessage.Text,
		}
		if submessage.From.ID == b.Client.Self.ID {
			m.Role = openai.ChatMessageRoleAssistant
		}
		chatMessages = append([]openai.ChatCompletionMessage{m}, chatMessages...)
		submessage = submessage.ReplyToMessage
	}
	resp, err := b.ai.CreateChatCompletion(context.TODO(), openai.ChatCompletionRequest{
		Model:       b.aiModel,
		Messages:    chatMessages,
		Temperature: 0.0,
	})

	content := ""
	if err != nil {
		content = "😕 AI 出错了，请稍后再试"
		b.logger.Errorf("ai request error: %s", err)
	} else {
		content = resp.Choices[0].Message.Content
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, content)
	msg.ReplyToMessageID = message.MessageID

	_, err = b.Client.Send(msg)
	if err != nil {
		b.logger.Errorf("openai reply error: %s", err)
	}
}
