#!/usr/bin/env python
# coding: utf-8

import os
import json
import logging
from datetime import datetime

from tweepy import Stream, StreamListener
from tweepy.models import Status

from yubari.lib.twitter import ttapi
from yubari.lib.qq import qqbot


logger = logging.getLogger(__name__)

KanColle_PROFILE_IMAGE_TMP = '/tmp/twitter_kancolle_staff_profile_image'
kancolle_profile_image = ""

TWITTERS = {
    "KanColle_STAFF": "294025417",
    "maesanpicture": "2381595966",
    "Strangestone": "93332575"
}


def load_profile_img():
    if not os.path.exists(KanColle_PROFILE_IMAGE_TMP):
        return
    global kancolle_profile_image
    with open(KanColle_PROFILE_IMAGE_TMP, 'r') as f:
        url = f.read()
        if url and url.startswith("https://"):
            kancolle_profile_image = url


def update_profile_img(img):
    global kancolle_profile_image
    if kancolle_profile_image == img:
        return
    kancolle_profile_image = img
    send_profile_image(img)
    logger.info("profile image changed to: %s", img)
    with open(KanColle_PROFILE_IMAGE_TMP, 'w') as f:
        f.write(img)


# TODO
def send_profile_image(img):
    qqbot.sendGroupMsg(img.replace("_normal", ""))


class MyStreamListener(StreamListener):
    def on_data(self, raw_data):
        data = json.loads(raw_data)
        if "event" in data:
            logger.debug("ignore event: %s", data["event"])
            return
        status = Status.parse(self.api, data)
        user = getattr(status, "user", None)
        if not user:
            logger.debug("empty user")
            return
        if user.id_str == TWITTERS["KanColle_STAFF"]:
            self.proceed_kancolle(status)
        elif user.id_str == TWITTERS["maesanpicture"]:
            self.proceed_samidare(status)
        elif user.id_str == TWITTERS["Strangestone"]:
            self.proceed_tawawa(status)
        else:
            logger.debug("ignore user: %s", user.name)

    def proceed_kancolle(self, status):
        tags = status.entities.get("hashtags", [])
        if "艦これ" not in tags:
            return
        logger.info("kancolle: %s", status.text.replace("\n", " "))
        user = status.user
        update_profile_img(user.profile_image_url_https)
        msg = [user.name]
        timestamp = int(int(status.timestamp_ms) / 1000)
        msg.append(datetime.fromtimestamp(timestamp).ctime())
        msg.append(status.text)
        qqbot.sendGroupMsg('\n'.join(msg))

    def proceed_samidare(self, status):
        tags = status.entities.get("hashtags", [])
        if "毎日五月雨" not in tags:
            return
        logger.info("maesan: %s", status.text.replace("\n", " "))
        medias = status.entities.get("media", [])
        for media in medias:
            logger.info("samidare: %s", media["media_url_https"])
            qqbot.sendSelfMsg(media["media_url_https"])

    def proceed_tawawa(self, status):
        if not status.text.startswith("月曜日のたわわ"):
            return
        logger.info("tawawa: %s", status.text.replace("\n", " "))
        medias = status.entities.get("media", [])
        for media in medias:
            logger.info("tawawa: %s", media["media_url_https"])
            qqbot.sendSelfMsg(media["media_url_https"])

    def on_error(self, code):
        logger.error("error: %s", code)


def run():
    load_profile_img()
    msl = Stream(auth=ttapi.auth, listener=MyStreamListener())
    #  msl.filter(track=[TAG_SAMIDARE], follow=[TWITTER_KanColle_STAFF, TWITTER_maesan])
    msl.filter(follow=TWITTERS.values())


if __name__ == '__main__':
    run()
