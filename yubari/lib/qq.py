#!/usr/bin/env python
# coding: utf-8

import os
import re
import logging
import base64
import requests
from requests.adapters import HTTPAdapter

from greenstalk.client import Client
from greenstalk.exceptions import NotFoundError

from yubari.consts import QQ_FACE_SEND, QQ_FACE_CODE, RE_QQ_FACE
from yubari.config import QQ_BOT, QQ_GROUP, QQ_ME, QQ_IMG_PATH

logger = logging.getLogger(__name__)

sess = requests.Session()
sess.mount('https://', HTTPAdapter(max_retries=3))


class QQBot(object):
    def __init__(self):
        self.client = Client(
            "localhost", 11300,
            use="%s(i)" % QQ_BOT,
            watch="%s(o)" % QQ_BOT)

    def _send(self, msg):
        self.client.put(msg)

    def _encode(self, msg):
        try:
            msg = re.sub(
                RE_QQ_FACE,
                lambda x: QQ_FACE_SEND.format(QQ_FACE_CODE.index(x.group(0))) if x.group(0) else x,
                msg)
        except Exception as e:
            logger.error("Failed replace face: %s", e)
        return base64.b64encode(msg.encode('GB18030')).decode()

    def _decode(self, msg):
        return base64.b64decode(msg).decode('GB18030')

    def _pull_img(self, url):
        filename = url.split('/')[-1]
        full_path = os.path.join(QQ_IMG_PATH, filename)
        if os.path.exists(full_path):
            logger.info("%s exists", filename)
        else:
            logger.info("--> Downloading %s", url)
            try:
                r = sess.get(url, stream=True, timeout=5)
                with open(full_path, 'wb') as f:
                    for chunk in r.iter_content(chunk_size=1024):
                        if chunk:
                            f.write(chunk)
            except Exception as e:
                logger.error("pull img failed: %s", e)
                return url
        return "[CQ:image,file={}]".format(filename)


    def sendGroupMsg(self, msg="", img=None):
        if img:
            msg += self._pull_img(img)
        if not msg:
            logger.error("send group msg empty")
        self._send("{} {} {}".format("sendGroupMsg", QQ_GROUP, self._encode(msg)))

    def sendPrivateMsg(self, qq, msg="", img=None):
        if img:
            msg += self._pull_img(img)
        if not msg:
            logger.error("send %s msg empty", qq)
        self._send("{} {} {}".format("sendPrivateMsg", qq, self._encode(msg)))

    def sendSelfMsg(self, msg="", img=None):
        self.sendPrivateMsg(QQ_ME, msg, img)

    def poll(self):
        while True:
            id_, body_ = self.client.reserve()
            if not id_ or not body_:
                continue
            body = body_.split()
            try:
                if body[0] == "eventPrivateMsg":
                    ret = dict(
                        event="PrivateMsg",
                        subtype=body[1],
                        time=body[2],
                        qq=body[3],
                        msg=self._decode(body[4]))
                elif body[0] == "eventGroupMsg":
                    ret = dict(
                        event="GroupMsg",
                        subtype=body[1],
                        time=body[2],
                        group=body[3],
                        qq=body[4],
                        anoymouse=body[5],
                        msg=self._decode(body[6]))
                else:
                    raise Exception("msg type not supported: %s" % body[0])
                yield ret
                self.client.delete(id_)
            except NotFoundError as e:
                logger.warning("msg not found to delete: {}".format(id_, e))
            except Exception as e:
                logger.error("failed to proceed msg [{}]: {}".format(body[4], e))
                self.client.bury(id_)


qqbot = QQBot()
