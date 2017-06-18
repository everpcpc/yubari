#!/usr/bin/env python
# coding: utf-8

import time
import logging

from yubari.config import QQ_GROUP, MENTION_NAME, QQ_ME
from yubari.lib.qq import qqbot


logger = logging.getLogger(__name__)


def check_mention_self(content):
    for word in MENTION_NAME:
        if word in content:
            return True
    return False


def run():
    continue_count = 0
    last_msg = ""
    last_call = 0
    for msg in qqbot.poll():
        now = int(time.time())
        if msg.get('event') == 'GroupMsg':
            if msg["qq"] == QQ_ME:
                last_call = now
            content = msg["msg"].strip()
            logger.info("(%s)[%s]:{%s}", msg["group"], msg["qq"], content)
            if check_mention_self(content):
                if now - last_call < 1200:
                    logger.info("called in last 30min")
                    continue
                call_msg = "呀呀呀，召唤一号机[CQ:at,qq=%s]" % QQ_ME
                qqbot.sendGroupMsg(call_msg)
                last_call = now
                continue
            if msg.get('group') == QQ_GROUP:
                if content != last_msg:
                    last_msg = content
                    continue_count = 0
                    continue
                if continue_count < 2:
                    continue_count += 1
                else:
                    logger.info("repeat: %s", content)
                    qqbot.sendGroupMsg(content)
                    continue_count = 0
        elif msg.get('event') == 'PrivateMsg':
            logger.info("[%s]:{%s}", msg["qq"], msg["msg"])
        else:
            logger.info(msg)


if __name__ == "__main__":
    run()
