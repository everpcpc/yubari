#!/usr/bin/env python
# coding: utf-8

import re


QQ_FACE_SEND = "[CQ:face,id={}]"
QQ_FACE_CODE = [
    "/惊讶", "/撇嘴", "/色", "/发呆", "/得意", "/流泪", "/害羞", "/闭嘴", "/睡", "/大哭",
    "/尴尬"
]

RE_QQ_FACE = re.compile(r"|".join(QQ_FACE_CODE))
