#!/usr/bin/env python
# coding: utf-8

import logging
from time import sleep
from multiprocessing import Process

from yubari.config import BOTS

logger = logging.getLogger(__name__)


def start_single(name, **kwargs):
    _bot = __import__("yubari.bots.%s" % name, globals(), locals(), ['run'], 0)
    _bot.run()


def start_all():
    processes = {}

    def start_subprocess(b):
        p_name = "%s_main" % b
        p = Process(
            target=start_single,
            name=p_name,
            args=(b,)
        )
        p.start()
        processes[b] = p
        logger.info("start process %d for %s.", p.pid, b)

    for b in BOTS:
        start_subprocess(b)

    while True:
        for b in processes:
            p = processes[b]
            sleep(1)
            if p.exitcode is None:
                if p.is_alive():
                    continue
                else:
                    logger.warn(
                        "%s not finished and not running, starting...", b)
                    start_subprocess(b)
            else:
                logger.warn("%s exited(%s), restarting...", b, p.exitcode)
                start_subprocess(b)


def main():
    start_all()


if __name__ == "__main__":
    main()
