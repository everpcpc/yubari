import html

from telethon import TelegramClient
from meilisearch import Client

# Use your own values from my.telegram.org
api_id = 1234567
api_hash = "111111111111111"

OLD_GROUP = None
GROUP = -100222222222
IDX = "supergroup--100222222222"

BOT_ID = 3333333

MEILI_URL = "http://localhost:7700/"
MEILI_TOKEN = "FFFFFFFFFFFF"


from local_config import *  # NOQA


def filter_message(msg):
    if not msg.from_id:
        return False
    if msg.from_id.user_id == BOT_ID:
        return False
    if not msg.message:
        return False
    if not msg.message.strip():
        return False
    if msg.message.startswith("/"):
        return False

    return True


def store_message(idx, client, group, is_old_group):
    count = 0
    real = 0
    bulk = []
    last = 0
    print(f"starting group {group}...")
    for message in client.iter_messages(group):
        count += 1
        if message.chat_id != group:
            print("!", message)
            continue
        if not filter_message(message):
            continue
        body = {
            "content": html.escape(message.message).replace("\n", " "),
            "date": int(message.date.timestamp()),
            "user": int(message.from_id.user_id),
            "id": -message.id if is_old_group else message.id,
        }
        # print("==> ", message.id, message.date, message.from_id.user_id, message.message)
        bulk.append(body)
        real += 1
        if len(bulk) == 1000:
            print(f"-> bulk add from {last} to {message.id}...")
            idx.add_documents(bulk, "id")
            bulk = []
            last = message.id

    if len(bulk) > 0:
        print(f"-> last bulk add from {last} to {message.id}...")
        idx.add_documents(bulk, "id")
    print(f"finished group {group}: {count} messages, indexed {real}")


# The first parameter is the .session file name (absolute paths allowed)
with TelegramClient("everpcpc", api_id, api_hash) as client:
    meili = Client(url=MEILI_URL, api_key=MEILI_TOKEN)
    idx = meili.index(IDX)
    store_message(idx, client, GROUP, False)
    if OLD_GROUP:
        store_message(idx, client, OLD_GROUP, True)
