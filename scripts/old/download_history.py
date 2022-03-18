import html

from telethon import TelegramClient
from elasticsearch import Elasticsearch

# Use your own values from my.telegram.org
api_id = 111111
api_hash = "00000000000000"

es = Elasticsearch("http://localhost:9200/")

OLD_GROUP = -111111111
GROUP = -10022222222
IDX = "supergroup--10022222222"

BOT_ID = 22222222

recreate_index = False
MAPPING = {
    "properties": {
        "content": {"type": "text", "analyzer": "ik_max_word", "search_analyzer": "ik_smart"},
        "id": {"type": "long"},
        "user": {"type": "long"},
        "date": {"type": "date"},
    }
}


def deleteElasticIndex():
    if es.indices.exists(index=IDX):
        es.indices.delete(index=IDX)


def ensureElasticIndex():
    if not es.indices.exists(index=IDX):
        es.indices.create(index=IDX)
        es.indices.put_mapping(index=IDX, body=MAPPING)


if recreate_index:
    deleteElasticIndex()
    ensureElasticIndex()


def filter_message(msg):
    if not message.from_id:
        return False
    if message.from_id.user_id == BOT_ID:
        return False
    if not msg.message:
        return False
    if not message.message.strip():
        return False
    if message.message.startswith("/"):
        return False

    return True


# The first parameter is the .session file name (absolute paths allowed)
with TelegramClient("everpcpc", api_id, api_hash) as client:
    count = 0

    for message in client.iter_messages(GROUP):
        count += 1

        if message.chat_id != GROUP:
            print("!", message)
            continue
        if not filter_message(message):
            continue

        print("==> ", message.id, message.date, message.from_id.user_id, message.message)
        es.index(
            index=IDX,
            id=message.id,
            body={
                "content": html.escape(message.message).replace("\n", " "),
                "user": int(message.from_id.user_id),
                "date": int(message.date.timestamp() * 1000),
                "id": message.id,
            },
        )

    for message in client.iter_messages(OLD_GROUP):
        count += 1
        if message.chat_id != OLD_GROUP:
            print("!", message)
            continue
        if not filter_message(message):
            continue
        print("==> ", message.id, message.date, message.from_id.user_id, message.message)
        es.index(
            index=IDX,
            id=-message.id,
            body={
                "content": html.escape(message.message).replace("\n", " "),
                "date": int(message.date.timestamp() * 1000),
                "user": int(message.from_id.user_id),
                "id": -message.id,
            },
        )
