#!/usr/bin/env python
# coding: utf-8

import os
import json
import logging

import requests
from requests.adapters import HTTPAdapter
from tweepy import Stream, StreamListener
from tweepy.models import Status

from yubari.config import TWITTER_IMG_PATH
from yubari.lib.twitter import ttapi


logger = logging.getLogger(__name__)

sess = requests.Session()
sess.mount('https://', HTTPAdapter(max_retries=3))


class PCStreamListener(StreamListener):
    def on_data(self, raw_data):
        data = json.loads(raw_data)
        if 'friends' in data:
            logger.debug('first time get friend list')
        elif 'event' in data:
            event = data['event']
            logger.debug('get event %s', event)
            event_fn = getattr(self, 'on_%s' % event, None)
            if event_fn is None:
                logger.warn('%s is not supported', event)
                return
            target = data.get('target_object')
            if not target:
                logger.warn('target is None on: %s', event)
                return
            status = Status.parse(self.api, target)
            event_fn(status)
        else:
            logger.debug('new timeline item')

    def on_favorite(self, status):
        self.process_image(status, 'download')

    def on_unfavorite(self, status):
        self.process_image(status, 'delete')

    def process_image(self, status, type_):
        medias = status.extended_entities.get("media", [])
        logger.info('[%s] %s medias', status.text.replace('\n', ' '), len(medias))
        for media in medias:
            if media['type'] == "photo":
                url = media['media_url_https']
                filename = url.split('/')[-1]
                full_path = os.path.join(TWITTER_IMG_PATH, filename)
                if type_ == 'download':
                    if os.path.exists(full_path):
                        logger.info("%s exists", filename)
                    else:
                        logger.info("--> Downloading %s", url)
                        r = sess.get(url, stream=True, timeout=5)
                        with open(full_path, 'wb') as f:
                            for chunk in r.iter_content(chunk_size=1024):
                                if chunk:
                                    f.write(chunk)
                elif type_ == 'delete':
                    if os.path.exists(full_path):
                        logger.info('--> Deleting %s', filename)
                        os.remove(full_path)
                    else:
                        logger.info('%s absent, ignore.', filename)
            else:
                logger.info('ignore media type %s', media['type'])

    def on_error(self, code):
        logger.error("error: %s", code)


def run():
    psl = Stream(auth=ttapi.auth, listener=PCStreamListener())
    psl.userstream(_with='user')


if __name__ == '__main__':
    run()
