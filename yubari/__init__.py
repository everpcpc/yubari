#!/usr/bin/env python
# coding: utf-8

import logging

logging.basicConfig(level=logging.INFO,
                    format='%(name)s - %(levelname)s - %(message)s')
logging.getLogger('requests.packages.urllib3.connectionpool').setLevel(logging.WARN)
