#!/usr/bin/env python
# coding: utf-8

import tweepy

from yubari.config import (TWITTER_CONSUMER_KEY,
                           TWITTER_CONSUMER_SECRET,
                           TWITTER_ACCESS_TOKEN,
                           TWITTER_ACCESS_TOKEN_SECRET)

_auth = tweepy.OAuthHandler(TWITTER_CONSUMER_KEY, TWITTER_CONSUMER_SECRET)
_auth.set_access_token(TWITTER_ACCESS_TOKEN, TWITTER_ACCESS_TOKEN_SECRET)

ttapi = tweepy.API(_auth)
