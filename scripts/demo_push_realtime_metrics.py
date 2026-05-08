#!/usr/bin/env python3
"""
Day mock metric len realtime (WebSocket) -> Kafka -> storage, de subscriber xem ban tin type=metric.

Dieu kien:
  - realtime-service + kafka + storage dang chay
  - JWT_SECRET trung .env realtime-service
  - User day du lieu phai dung JWT cua chinh user do (server chi cho push user_id = sub)

Vi du (Alice E2E day, Bob xem):
  Terminal 1 - xem:
    python scripts/demo_push_realtime_metrics.py --mode watch --user-id bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb \\
      --target-id aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa --group-id 11111111-1111-4111-8111-111111111111

  Terminal 2 - day (mac dinh moi 3s, 0=lap vo han):
    python scripts/demo_push_realtime_metrics.py --mode publish --user-id aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa \\
      --count 0 --interval 3

Cai: pip install -r scripts/e2e_requirements.txt

Bien moi truong:
  REALTIME_WS_URL  mac dinh ws://127.0.0.1:5001/ws
  JWT_SECRET       mac dinh healthmate_jwt_secret_dev
"""

from __future__ import annotations

import argparse
import json
import os
import random
import sys
import time
import uuid
from typing import Any
from urllib.parse import quote

import jwt

try:
    from websocket import WebSocketException, create_connection
except ImportError:
    print("Need: pip install websocket-client", file=sys.stderr)
    raise SystemExit(1)

DEFAULT_GROUP = "11111111-1111-4111-8111-111111111111"


def mint_token(sub: uuid.UUID, email: str, secret: str) -> str:
    now = int(time.time())
    payload: dict[str, Any] = {
        "sub": str(sub),
        "email": email,
        "type": "access",
        "role": "user",
        "iat": now,
        "exp": now + 7200,
    }
    return jwt.encode(payload, secret, algorithm="HS256")


def ws_url(base: str, token: str) -> str:
    sep = "&" if "?" in base else "?"
    return f"{base}{sep}token={quote(token, safe='')}"


def default_email_for_uuid(uid: uuid.UUID) -> str:
    s = str(uid)
    if s.startswith("aaaaaaaa"):
        return "e2e_share_alice@healthmate.test"
    if s.startswith("bbbbbbbb"):
        return "e2e_share_bob@healthmate.test"
    if s.startswith("cccccccc"):
        return "e2e_share_charlie@healthmate.test"
    return f"demo_{s[:8]}@healthmate.test"


def subscribe_msg(target: uuid.UUID, metric: str, group_id: str) -> str:
    return json.dumps(
        {
            "action": "subscribe",
            "items": [
                {
                    "target_user_id": str(target),
                    "metric_type": metric,
                    "group_id": group_id,
                }
            ],
        }
    )


def push_msg(user_id: uuid.UUID, metric: str, value: float) -> str:
    ts = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
    return json.dumps(
        {
            "user_id": str(user_id),
            "metric_type": metric,
            "value": value,
            "timestamp": ts,
        }
    )


def run_watch(
    base: str,
    secret: str,
    watcher_id: uuid.UUID,
    target_id: uuid.UUID,
    group_id: str,
    metrics: list[str],
) -> None:
    email = default_email_for_uuid(watcher_id)
    tok = mint_token(watcher_id, email, secret)
    try:
        ws = create_connection(ws_url(base, tok), timeout=20)
    except WebSocketException as e:
        print(f"WS connect failed: {e}", file=sys.stderr)
        raise SystemExit(1)
    try:
        for m in metrics:
            ws.send(subscribe_msg(target_id, m, group_id))
            raw = ws.recv()
            print(f"subscribe {m}: {raw!r}")
        print("Listening for type=metric (Ctrl+C to stop)...")
        while True:
            raw = ws.recv()
            if isinstance(raw, bytes):
                raw = raw.decode("utf-8", errors="replace")
            try:
                data = json.loads(raw)
            except json.JSONDecodeError:
                print(raw)
                continue
            if data.get("type") == "metric":
                print(json.dumps(data, ensure_ascii=True))
            else:
                print(raw)
    except KeyboardInterrupt:
        print("\nStopped.")
    finally:
        try:
            ws.close()
        except Exception:
            pass


def run_publish(
    base: str,
    secret: str,
    publisher_id: uuid.UUID,
    metrics: list[str],
    interval: float,
    count: int,
) -> None:
    email = default_email_for_uuid(publisher_id)
    tok = mint_token(publisher_id, email, secret)
    try:
        ws = create_connection(ws_url(base, tok), timeout=20)
    except WebSocketException as e:
        print(f"WS connect failed: {e}", file=sys.stderr)
        raise SystemExit(1)

    hr = 72.0
    steps = 4000.0
    cal = 150.0
    n = 0
    try:
        while count == 0 or n < count:
            n += 1
            for m in metrics:
                if m == "heart_rate":
                    hr = max(55.0, min(110.0, hr + random.uniform(-3, 3)))
                    ws.send(push_msg(publisher_id, m, hr))
                elif m == "steps_count":
                    steps += random.uniform(50, 300)
                    ws.send(push_msg(publisher_id, m, steps))
                elif m == "calories_burned":
                    cal += random.uniform(0.5, 4.0)
                    ws.send(push_msg(publisher_id, m, cal))
                elif m == "blood_pressure":
                    ws.send(push_msg(publisher_id, m, 110 + random.uniform(0, 25)))
                elif m == "spo2":
                    ws.send(push_msg(publisher_id, m, 95 + random.uniform(0, 4)))
                else:
                    ws.send(push_msg(publisher_id, m, random.uniform(10, 100)))
                time.sleep(0.05)
            maybe = ws.recv()
            if isinstance(maybe, bytes):
                maybe = maybe.decode("utf-8", errors="replace")
            if maybe:
                print(f"[{n}] ack/side: {maybe[:200]}")
            time.sleep(max(0.5, interval))
    except KeyboardInterrupt:
        print("\nStopped.")
    finally:
        try:
            ws.close()
        except Exception:
            pass


def main() -> None:
    p = argparse.ArgumentParser(description="Demo push/watch realtime metrics via WebSocket")
    p.add_argument(
        "--mode",
        choices=("publish", "watch"),
        required=True,
    )
    p.add_argument("--user-id", type=uuid.UUID, required=True, help="JWT sub + WS identity")
    p.add_argument(
        "--target-id",
        type=uuid.UUID,
        help="For watch: whose metrics to subscribe (e.g. Alice)",
    )
    p.add_argument(
        "--group-id",
        default=os.environ.get("DEMO_GROUP_ID", DEFAULT_GROUP),
        help="Group UUID for subscribe",
    )
    p.add_argument(
        "--metrics",
        default="heart_rate,steps_count,calories_burned",
        help="Comma-separated metric types",
    )
    p.add_argument("--interval", type=float, default=3.0, help="Seconds between publish rounds")
    p.add_argument(
        "--count",
        type=int,
        default=0,
        help="Publish rounds (0 = infinite)",
    )
    args = p.parse_args()

    base = os.environ.get("REALTIME_WS_URL", "ws://127.0.0.1:5001/ws").rstrip("/")
    if not base.endswith("/ws"):
        base = base.rstrip("/") + "/ws"
    secret = os.environ.get("JWT_SECRET", "healthmate_jwt_secret_dev").strip('"')

    metrics = [m.strip() for m in args.metrics.split(",") if m.strip()]

    if args.mode == "watch":
        if args.target_id is None:
            print("watch mode requires --target-id", file=sys.stderr)
            raise SystemExit(2)
        run_watch(base, secret, args.user_id, args.target_id, args.group_id, metrics)
    else:
        run_publish(base, secret, args.user_id, metrics, args.interval, args.count)


if __name__ == "__main__":
    main()
