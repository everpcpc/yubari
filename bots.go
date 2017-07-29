package main

func qqWatch(messages chan map[string]string) {
	ignoreMap := make(map[string]struct{})
	for _, q := range qqBot.Cfg.QQIgnore {
		ignoreMap[q] = struct{}{}
	}

	for msg := range messages {
		switch msg["event"] {
		case "PrivateMsg":
			logger.Infof("[%s]:{%s}", msg["qq"], msg["msg"])
		case "GroupMsg":
			if _, ok := ignoreMap[msg["qq"]]; ok {
				logger.Debugf("Ignore (%s)[%s]:{%s}", msg["group"], msg["qq"], msg["msg"])
				continue
			}
			go qqBot.NoticeMention(msg["msg"], msg["group"])
			go qqBot.CheckRepeat(msg["msg"], msg["group"])
			logger.Infof("(%s)[%s]:{%s}", msg["group"], msg["qq"], msg["msg"])
		default:
			logger.Info(msg)
		}
	}
}

func twitterTrack() {
}

func twitterPics() {
}
