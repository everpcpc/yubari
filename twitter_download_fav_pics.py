#!/usr/bin/env python3
# coding: utf-8

import os
import json
import logging

import requests
from requests.adapters import HTTPAdapter

import tweepy
from tweepy.models import Status

config_file = '/home/everpcpc/config/twitter.json'

with open(config_file, 'r', encoding='utf-8') as f:
    conf = json.load(f)

DOWNLOAD_PATH = conf['DOWNLOAD_PATH']

auth = tweepy.OAuthHandler(conf['CONSUMER_KEY'], conf['CONSUMER_SECRET'])
auth.set_access_token(conf['ACCESS_TOKEN'], conf['ACCESS_TOKEN_SECRET'])

logger = logging.getLogger('twitter')
logging.basicConfig(level=logging.INFO,
                    format='%(name)s - %(levelname)s - %(message)s')
logging.getLogger('requests.packages.urllib3.connectionpool').setLevel(logging.WARN)

sess = requests.Session()
sess.mount('https://', HTTPAdapter(max_retries=3))
api = tweepy.API(auth)


class PCStreamListener(tweepy.StreamListener):
    def on_data(self, raw_data):
        data = json.loads(raw_data)
        if 'event' in data:
            event = data['event']
            logger.info('get event %s' % event)
            event_fn = getattr(self, 'on_%s' % event, None)
            status = Status.parse(self.api, data['target_object'])
            if event_fn is None:
                logger.warn('%s is not supported' % event)
            else:
                if event_fn(status) is False:
                    return False
        else:
            logger.info('new timeline item')

    def on_favorite(self, status):
        self.process_image(status, 'download')

    def on_unfavorite(self, status):
        self.process_image(status, 'delete')

    def process_image(self, status, type_):
        medias = status.entities.get('media', [])
        logger.info('[%s]', status.text.replace('\n', ' '))
        for media in medias:
            if media['type'] == "photo":
                url = media['media_url_https']
                filename = url.split('/')[-1]
                full_path = os.path.join(DOWNLOAD_PATH, filename)
                if type_ == 'download':
                    if os.path.exists(full_path):
                        logger.info("%s exists" % filename)
                    else:
                        logger.info("--> Downloading %s" % url)
                        r = sess.get(url, stream=True, timeout=5)
                        with open(full_path, 'wb') as f:
                            for chunk in r.iter_content(chunk_size=1024):
                                if chunk:
                                    f.write(chunk)
                elif type_ == 'delete':
                    if os.path.exists(full_path):
                        logger.info('--> Deleting %s' % filename)
                        os.remove(full_path)
                    else:
                        logger.info('%s absent, ignore.' % filename)


if __name__ == '__main__':
    psl = tweepy.Stream(auth=api.auth, listener=PCStreamListener())
    psl.userstream(_with='everpcpc', async=True)
