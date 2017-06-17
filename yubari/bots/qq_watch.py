#!/usr/bin/env python
# coding: utf-8

import logging

from yubari.config import QQ_GROUP, MENTION_NAME
from yubari.lib.qq import qqbot


logger = logging.getLogger(__name__)


def run():
    continue_count = 0
    last_msg = ""
    for msg in qqbot.poll():
        logger.info(msg)
        content = msg.get('msg')
        for word in MENTION_NAME:
            if word in content:
                qqbot.sendSelfMsg(content)
        if msg.get('event') == 'GroupMsg':
            if msg.get('group') == QQ_GROUP:
                if content != last_msg:
                    last_msg = content
                    continue_count = 0
                    continue
                if continue_count < 2:
                    continue_count += 1
                else:
                    logger.info("repeat: %s", content)
                    qqbot.sendSelfMsg(content)
                    continue_count = 0


if __name__ == "__main__":
    run()
