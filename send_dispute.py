#!/usr/bin/env python3
"""
Send a dispute notification with photo to a Telegram chat.

Usage:
    python send_dispute.py \
        --chat_id="-100123456789" \
        --transaction="d84cd4bc-9777-437b-a2dc-591e52847cfd" \
        --photo="/path/to/evidence.jpg"

    # Multiple photos:
    python send_dispute.py \
        --chat_id="-100123456789" \
        --transaction="d84cd4bc-..." \
        --photo="/path/1.jpg" \
        --photo="/path/2.jpg"

Environment variable BOT_TOKEN must be set (or use --token flag).
"""

import argparse
import os
import sys
import urllib.request
import urllib.parse
import json
from pathlib import Path


def parse_args():
    parser = argparse.ArgumentParser(description="Send dispute notification to Telegram")
    parser.add_argument("--chat_id", required=True, help="Telegram chat/group/channel ID")
    parser.add_argument("--transaction", required=True, help="Transaction UUID")
    parser.add_argument(
        "--photo",
        action="append",
        dest="photos",
        metavar="PATH",
        required=True,
        help="Path to evidence photo (repeat flag for multiple)",
    )
    parser.add_argument(
        "--token",
        default=os.environ.get("BOT_TOKEN"),
        help="Telegram bot token (default: $BOT_TOKEN)",
    )
    parser.add_argument(
        "--caption",
        default=None,
        help="Override default caption text",
    )
    return parser.parse_args()


def tg_api(token: str, method: str, **fields) -> dict:
    url = f"https://api.telegram.org/bot{token}/{method}"
    data = urllib.parse.urlencode(fields).encode()
    req = urllib.request.Request(url, data=data)
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.loads(resp.read())


def send_photo(token: str, chat_id: str, photo_path: str, caption: str | None = None) -> dict:
    """Send a single photo via multipart/form-data."""
    import http.client
    import mimetypes
    import uuid

    boundary = uuid.uuid4().hex
    photo_bytes = Path(photo_path).read_bytes()
    mime = mimetypes.guess_type(photo_path)[0] or "image/jpeg"
    filename = Path(photo_path).name

    parts = []
    # chat_id field
    parts.append(
        f'--{boundary}\r\n'
        f'Content-Disposition: form-data; name="chat_id"\r\n\r\n'
        f'{chat_id}\r\n'
    )
    # caption field
    if caption:
        parts.append(
            f'--{boundary}\r\n'
            f'Content-Disposition: form-data; name="caption"\r\n'
            f'Content-Type: text/plain; charset=utf-8\r\n\r\n'
            f'{caption}\r\n'
        )
    # parse_mode
    parts.append(
        f'--{boundary}\r\n'
        f'Content-Disposition: form-data; name="parse_mode"\r\n\r\n'
        f'HTML\r\n'
    )

    header = "".join(parts).encode("utf-8")
    photo_part = (
        f'--{boundary}\r\n'
        f'Content-Disposition: form-data; name="photo"; filename="{filename}"\r\n'
        f'Content-Type: {mime}\r\n\r\n'
    ).encode("utf-8")
    footer = f'\r\n--{boundary}--\r\n'.encode("utf-8")

    body = header + photo_part + photo_bytes + footer

    conn = http.client.HTTPSConnection("api.telegram.org", timeout=30)
    conn.request(
        "POST",
        f"/bot{token}/sendPhoto",
        body=body,
        headers={"Content-Type": f"multipart/form-data; boundary={boundary}"},
    )
    resp = conn.getresponse()
    result = json.loads(resp.read())
    conn.close()
    return result


def send_media_group(token: str, chat_id: str, photos: list[str], caption: str) -> dict:
    """Send multiple photos as an album (media group)."""
    import http.client
    import mimetypes
    import uuid

    boundary = uuid.uuid4().hex
    media = []
    file_fields = {}

    for i, path in enumerate(photos):
        attach_name = f"photo{i}"
        entry = {"type": "photo", "media": f"attach://{attach_name}"}
        if i == 0:
            entry["caption"] = caption
            entry["parse_mode"] = "HTML"
        media.append(entry)
        file_fields[attach_name] = path

    # Build multipart body
    def field(name: str, value: str) -> bytes:
        return (
            f'--{boundary}\r\nContent-Disposition: form-data; name="{name}"\r\n\r\n{value}\r\n'
        ).encode("utf-8")

    body = field("chat_id", chat_id) + field("media", json.dumps(media))

    for attach_name, path in file_fields.items():
        photo_bytes = Path(path).read_bytes()
        mime = mimetypes.guess_type(path)[0] or "image/jpeg"
        filename = Path(path).name
        body += (
            f'--{boundary}\r\n'
            f'Content-Disposition: form-data; name="{attach_name}"; filename="{filename}"\r\n'
            f'Content-Type: {mime}\r\n\r\n'
        ).encode("utf-8") + photo_bytes + b'\r\n'

    body += f'--{boundary}--\r\n'.encode("utf-8")

    conn = http.client.HTTPSConnection("api.telegram.org", timeout=60)
    conn.request(
        "POST",
        f"/bot{token}/sendMediaGroup",
        body=body,
        headers={"Content-Type": f"multipart/form-data; boundary={boundary}"},
    )
    resp = conn.getresponse()
    result = json.loads(resp.read())
    conn.close()
    return result


def main():
    args = parse_args()

    if not args.token:
        print("ERROR: BOT_TOKEN not set. Use --token or export BOT_TOKEN=<your_token>", file=sys.stderr)
        sys.exit(1)

    # Validate photos exist
    for p in args.photos:
        if not Path(p).is_file():
            print(f"ERROR: photo file not found: {p}", file=sys.stderr)
            sys.exit(1)

    caption = args.caption or f"<b>Новый спор по сделке:</b>\n<code>{args.transaction}</code>"

    try:
        if len(args.photos) == 1:
            result = send_photo(args.token, args.chat_id, args.photos[0], caption)
        else:
            result = send_media_group(args.token, args.chat_id, args.photos, caption)
    except Exception as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        sys.exit(1)

    if not result.get("ok"):
        print(f"Telegram API error: {result}", file=sys.stderr)
        sys.exit(1)

    print(f"OK: message sent to {args.chat_id} (transaction={args.transaction})")


if __name__ == "__main__":
    main()
