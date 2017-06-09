#!/usr/bin/env python
# coding: utf-8

from setuptools import setup, find_packages


setup(
    name='Yubari',
    version=0.1,
    description="",
    long_description=open("README.md").read(),
    classifiers=[
        "Programming Language :: Python",
    ],
    keywords='',
    author='everpcpc',
    author_email='git@everpcpc.com',
    license='everpcpc',
    packages=find_packages(exclude=['ez_setup', 'examples*', 'tests*']),
    include_package_data=True,
    zip_safe=False,
    entry_points='''
        [console_scripts]
        yubari = yubari.app:main
    ''',
    install_requires=[
        'setuptools',
        'requests',
        'tweepy',
        'greenstalk',
    ],
)
