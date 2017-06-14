#!/usr/bin/env python
# coding: utf-8

import re
import base64

from greenstalk.client import Client

from yubari.consts import QQ_FACE_SEND, QQ_FACE_CODE, RE_QQ_FACE
from yubari.config import QQ_BOT, QQ_GROUP, QQ_ME


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
        return base64.b64encode(msg.encode('gbk')).decode()

    def _decode(self, msg):
        return base64.b64decode(msg).decode('gbk')

    def sendGroupMsg(self, msg):
        self._send("{} {} {}".format("sendGroupMsg", QQ_GROUP, self._encode(msg)))

    def sendPrivateMsg(self, qq, msg):
        self._send("{} {} {}".format("sendPrivateMsg", qq, self._encode(msg)))

    def sendSelfMsg(self, msg):
        self.sendPrivateMsg(QQ_ME, msg)

    def poll(self):
        while True:
            id_, body_ = self.client.reserve()
            if not id_ or not body_:
                continue
            body = body_.split()
            try:
                if body[0] == "eventPrivateMsg":
                    yield dict(
                        event="PrivateMsg",
                        subtype=body[1],
                        time=body[2],
                        qq=body[3],
                        msg=self._decode(body[4]))
                    self.client.delete(id_)
                elif body[0] == "eventGroupMsg":
                    yield dict(
                        event="GroupMsg",
                        subtype=body[1],
                        time=body[2],
                        group=body[3],
                        qq=body[4],
                        anoymouse=body[5],
                        msg=self._decode(body[6]))
                else:
                    raise Exception("msg type not supported: %s" % body[0])
                self.client.delete(id_)
            except Exception as e:
                logging.error("failed to proceed msg [{}]: {}".format(body[4], e))
                self.client.bury(id_)


qqbot = QQBot()
