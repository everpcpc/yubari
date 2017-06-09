#!/usr/bin/env python
# coding: utf-8

import logging

from yubari.lib.qq import qqbot


logger = logging.getLogger(__name__)


def run():
    for msg in qqbot.poll():
        logger.info(msg)


if __name__ == "__main__":
    run()
