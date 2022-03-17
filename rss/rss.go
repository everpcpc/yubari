package rss

import (
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-redis/redis"
	"github.com/mmcdole/gofeed"
	"github.com/sirupsen/logrus"
)

type bot struct {
  redis *redis.Client
  logger *logrus.logger
  target string
  ttl int
}
