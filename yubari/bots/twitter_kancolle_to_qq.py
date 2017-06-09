#!/usr/bin/env python
# coding: utf-8

import os
import json
import logging
from datetime import datetime

from tweepy import Stream, StreamListener
from tweepy.models import Status

from yubari.config import TWITTER_KanColle_STAFF
from yubari.lib.twitter import ttapi
from yubari.lib.qq import qqbot


logger = logging.getLogger(__name__)


PROFILE_IMAGE_TMP = '/tmp/twitter_kancolle_staff_profile_image'
profile_image = ""


def load_profile_img():
    if not os.path.exists(PROFILE_IMAGE_TMP):
        return
    global profile_image
    with open(PROFILE_IMAGE_TMP, 'r') as f:
        url = f.read()
        if url and url.startswith("https://"):
            profile_image = url


def update_profile_img(img):
    global profile_image
    if profile_image == img:
        return
    profile_image = img
    send_image(profile_image)
    logger.info("profile image changed to: %s", profile_image)
    with open(PROFILE_IMAGE_TMP, 'w') as f:
        f.write(img)


# TODO
def send_image(img):
    qqbot.sendGroupMsg(img.replace("_normal", ""))


class KancolleStreamListener(StreamListener):
    def on_data(self, raw_data):
        global profile_image

        data = json.loads(raw_data)
        if "event" in data:
            logger.debug("ignore event: %s", data["event"])
            return

        status = Status.parse(self.api, data)
        user = status.user
        if not user:
            logger.debug("empty user")
            return
        # not user msg
        if user.id_str != TWITTER_KanColle_STAFF:
            logger.debug("ignore msg from: %s", user.name)
            return
        update_profile_img(user.profile_image_url_https)

        msg = [user.name]
        timestamp = int(int(status.timestamp_ms) / 1000)
        msg.append(datetime.fromtimestamp(timestamp).ctime())
        logger.info("recv: %s", status.text.replace("\n", " "))
        msg.append(status.text)
        qqbot.sendGroupMsg('\n'.join(msg))

    def on_error(self, code):
        logger.error("error: %s", code)


def run():
    load_profile_img()
    ksl = Stream(auth=ttapi.auth, listener=KancolleStreamListener())
    ksl.filter(follow=[TWITTER_KanColle_STAFF])


if __name__ == '__main__':
    run()
