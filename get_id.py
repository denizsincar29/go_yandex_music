# This script is needed because yandex music go does not have a way to get the id from the token only.
# If you don't have a token, go check @MusoadBot on telegram.
#
# pip install -U yandex-music
# 
from yandex_music import Client

token = input("Enter your Yandex Music token: ")

client = Client(token).init()
uid = client.me["account"]["uid"]
with open(".env", "w", encoding="UTF-8") as f:
    f.write(f"YA_MUSIC_ID={uid}")
    f.write(f"\nYA_MUSIC_TOKEN={token}")