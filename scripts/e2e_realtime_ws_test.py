#!/usr/bin/env python3
"""
E2E realtime: WebSocket + Kafka + quyền sharing_permissions (giống hub broadcast).

Điều kiện:
  - docker compose: timescaledb, kafka, zookeeper, realtime-service (và api-gateway nếu test qua nginx).
  - Đã nạp scripts/e2e_sharing_permissions_seed.sql (cùng UUID Alice/Bob/Charlie/Eve/Frank/nhóm).

Luồng:
  1) Clients mở WS, gửi subscribe (Alice + metric_type + group_id).
  2) Alice mở WS, push JSON chỉ số → realtime publish Kafka → consumer → hub.
  3) Hub gọi GetMetricWatchers(Alice, metric); chỉ client được phép mới nhận message type=metric.

Cài: pip install -r scripts/e2e_requirements.txt

Biến môi trường:
  REALTIME_WS_URL   mặc định ws://127.0.0.1:5001/ws
  JWT_SECRET        mặc định healthmate_jwt_secret_dev
"""

from __future__ import annotations

import json
import os
import sys
import time
import uuid
from typing import Any
from urllib.parse import quote

import jwt

try:
    from websocket import create_connection, WebSocketException
except ImportError:
    print("Need: pip install websocket-client", file=sys.stderr)
    raise SystemExit(1)

ALICE = uuid.UUID("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
BOB = uuid.UUID("bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb")
CHARLIE = uuid.UUID("cccccccc-cccc-4ccc-8ccc-cccccccccccc")
EVE = uuid.UUID("eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee")
FRANK = uuid.UUID("ffffffff-ffff-4fff-8fff-ffffffffffff")
GROUP = "11111111-1111-4111-8111-111111111111"


def mint_token(sub: uuid.UUID, email: str, secret: str) -> str:
    now = int(time.time())
    payload: dict[str, Any] = {
        "sub": str(sub),
        "email": email,
        "type": "access",
        "role": "user",
        "iat": now,
        "exp": now + 3600,
    }
    return jwt.encode(payload, secret, algorithm="HS256")


def ws_url(base: str, token: str) -> str:
    sep = "&" if "?" in base else "?"
    return f"{base}{sep}token={quote(token, safe='')}"


def recv_json(ws: Any, timeout: float = 8.0) -> dict[str, Any] | None:
    ws.settimeout(timeout)
    try:
        raw = ws.recv()
    except Exception:
        return None
    if isinstance(raw, bytes):
        raw = raw.decode("utf-8")
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        return None


def subscribe(ws: Any, target: uuid.UUID, metric: str, group_id: str) -> None:
    msg = {
        "action": "subscribe",
        "items": [
            {
                "target_user_id": str(target),
                "metric_type": metric,
                "group_id": group_id,
            }
        ],
    }
    ws.send(json.dumps(msg))


def push_metric(ws: Any, user_id: uuid.UUID, metric: str, value: float) -> None:
    ts = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
    payload = {
        "user_id": str(user_id),
        "metric_type": metric,
        "value": value,
        "timestamp": ts,
    }
    ws.send(json.dumps(payload))


def wait_for_metric(ws: Any, want_type: str, timeout: float = 12.0) -> bool:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        ws.settimeout(max(0.5, deadline - time.monotonic()))
        try:
            raw = ws.recv()
        except Exception:
            continue
        if isinstance(raw, bytes):
            raw = raw.decode("utf-8")
        try:
            data = json.loads(raw)
        except json.JSONDecodeError:
            continue
        if data.get("type") == "metric":
            p = data.get("payload") or {}
            if p.get("metric_type") == want_type:
                return True
    return False


def no_metric_before_deadline(ws: Any, seconds: float = 5.0) -> bool:
    """True if no type=metric received within window."""
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        ws.settimeout(max(0.3, deadline - time.monotonic()))
        try:
            raw = ws.recv()
        except Exception:
            continue
        if isinstance(raw, bytes):
            raw = raw.decode("utf-8")
        try:
            data = json.loads(raw)
        except json.JSONDecodeError:
            continue
        if data.get("type") == "metric":
            return False
    return True


def ack_subscribes(subscribers: list[Any]) -> bool:
    for ws in subscribers:
        if recv_json(ws) is None:
            return False
    return True


def main() -> int:
    base = os.environ.get("REALTIME_WS_URL", "ws://127.0.0.1:5001/ws").rstrip("/")
    if not base.endswith("/ws"):
        base = base.rstrip("/") + "/ws"
    secret = os.environ.get("JWT_SECRET", "healthmate_jwt_secret_dev").strip('"')

    tok_alice = mint_token(ALICE, "e2e_share_alice@healthmate.test", secret)
    tok_bob = mint_token(BOB, "e2e_share_bob@healthmate.test", secret)
    tok_charlie = mint_token(CHARLIE, "e2e_share_charlie@healthmate.test", secret)
    tok_eve = mint_token(EVE, "e2e_share_eve@healthmate.test", secret)
    tok_frank = mint_token(FRANK, "e2e_share_frank@healthmate.test", secret)

    print(f"Realtime WS: {base}\n")

    failed = 0

    # --- Case A: heart_rate — Bob OK, Charlie không được nhận ---
    try:
        wb = create_connection(ws_url(base, tok_bob), timeout=15)
        wc = create_connection(ws_url(base, tok_charlie), timeout=15)
        wa = create_connection(ws_url(base, tok_alice), timeout=15)
    except WebSocketException as e:
        print(f"FAIL: WS connect error (is realtime-service + Kafka up?): {e}")
        return 1

    try:
        subscribe(wb, ALICE, "heart_rate", GROUP)
        subscribe(wc, ALICE, "heart_rate", GROUP)
        if not ack_subscribes([wb, wc]):
            print("FAIL: no response after subscribe (heart_rate)")
            failed += 1
        else:
            push_metric(wa, ALICE, "heart_rate", 82.0)
            time.sleep(0.5)
            if not wait_for_metric(wb, "heart_rate", timeout=15.0):
                print("FAIL: Bob must receive heart_rate")
                failed += 1
            else:
                print("OK: Bob received heart_rate realtime")
            if not no_metric_before_deadline(wc, seconds=6.0):
                print("FAIL: Charlie must NOT receive heart_rate")
                failed += 1
            else:
                print("OK: Charlie did not receive heart_rate (expected)")
    finally:
        for c in (wb, wc, wa):
            try:
                c.close()
            except Exception:
                pass

    # --- Case B: steps_count — Bob và Charlie đều nhận ---
    try:
        wb2 = create_connection(ws_url(base, tok_bob), timeout=15)
        wc2 = create_connection(ws_url(base, tok_charlie), timeout=15)
        wa2 = create_connection(ws_url(base, tok_alice), timeout=15)
    except WebSocketException as e:
        print(f"FAIL (steps round): {e}")
        return 1

    try:
        subscribe(wb2, ALICE, "steps_count", GROUP)
        subscribe(wc2, ALICE, "steps_count", GROUP)
        if not ack_subscribes([wb2, wc2]):
            print("FAIL: no ack after subscribe steps_count")
            failed += 1
        else:
            push_metric(wa2, ALICE, "steps_count", 5000.0)
            time.sleep(0.5)
            if not wait_for_metric(wb2, "steps_count", timeout=15.0):
                print("FAIL: Bob must receive steps_count")
                failed += 1
            else:
                print("OK: Bob received steps_count realtime")
            if not wait_for_metric(wc2, "steps_count", timeout=15.0):
                print("FAIL: Charlie must receive steps_count")
                failed += 1
            else:
                print("OK: Charlie received steps_count realtime")
    finally:
        for c in (wb2, wc2, wa2):
            try:
                c.close()
            except Exception:
                pass

    # --- Case C: blood_pressure — Bob, Frank, Eve nhận; Charlie không ---
    try:
        wb3 = create_connection(ws_url(base, tok_bob), timeout=15)
        wf3 = create_connection(ws_url(base, tok_frank), timeout=15)
        we3 = create_connection(ws_url(base, tok_eve), timeout=15)
        wc3 = create_connection(ws_url(base, tok_charlie), timeout=15)
        wa3 = create_connection(ws_url(base, tok_alice), timeout=15)
    except WebSocketException as e:
        print(f"FAIL (blood_pressure connect): {e}")
        return 1

    try:
        subscribe(wb3, ALICE, "blood_pressure", GROUP)
        subscribe(wf3, ALICE, "blood_pressure", GROUP)
        subscribe(we3, ALICE, "blood_pressure", GROUP)
        subscribe(wc3, ALICE, "blood_pressure", GROUP)
        if not ack_subscribes([wb3, wf3, we3, wc3]):
            print("FAIL: no ack after subscribe blood_pressure")
            failed += 1
        else:
            push_metric(wa3, ALICE, "blood_pressure", 119.0)
            time.sleep(0.5)
            for label, ws in ("Bob", wb3), ("Frank", wf3), ("Eve", we3):
                if not wait_for_metric(ws, "blood_pressure", timeout=15.0):
                    print(f"FAIL: {label} must receive blood_pressure")
                    failed += 1
                else:
                    print(f"OK: {label} received blood_pressure realtime")
            if not no_metric_before_deadline(wc3, seconds=6.0):
                print("FAIL: Charlie must NOT receive blood_pressure")
                failed += 1
            else:
                print("OK: Charlie did not receive blood_pressure (expected)")
    finally:
        for c in (wb3, wf3, we3, wc3, wa3):
            try:
                c.close()
            except Exception:
                pass

    # --- Case D: calories_burned — Bob, Frank nhận; Charlie và Eve không ---
    try:
        wb4 = create_connection(ws_url(base, tok_bob), timeout=15)
        wf4 = create_connection(ws_url(base, tok_frank), timeout=15)
        we4 = create_connection(ws_url(base, tok_eve), timeout=15)
        wc4 = create_connection(ws_url(base, tok_charlie), timeout=15)
        wa4 = create_connection(ws_url(base, tok_alice), timeout=15)
    except WebSocketException as e:
        print(f"FAIL (calories connect): {e}")
        return 1

    try:
        subscribe(wb4, ALICE, "calories_burned", GROUP)
        subscribe(wf4, ALICE, "calories_burned", GROUP)
        subscribe(we4, ALICE, "calories_burned", GROUP)
        subscribe(wc4, ALICE, "calories_burned", GROUP)
        if not ack_subscribes([wb4, wf4, we4, wc4]):
            print("FAIL: no ack after subscribe calories_burned")
            failed += 1
        else:
            push_metric(wa4, ALICE, "calories_burned", 210.0)
            time.sleep(0.5)
            for label, ws in ("Bob", wb4), ("Frank", wf4):
                if not wait_for_metric(ws, "calories_burned", timeout=15.0):
                    print(f"FAIL: {label} must receive calories_burned")
                    failed += 1
                else:
                    print(f"OK: {label} received calories_burned realtime")
            for label, ws in ("Charlie", wc4), ("Eve", we4):
                if not no_metric_before_deadline(ws, seconds=6.0):
                    print(f"FAIL: {label} must NOT receive calories_burned")
                    failed += 1
                else:
                    print(f"OK: {label} did not receive calories_burned (expected)")
    finally:
        for c in (wb4, wf4, we4, wc4, wa4):
            try:
                c.close()
            except Exception:
                pass

    if failed:
        print(f"\nResult: {failed} failed.")
        return 1
    print("\nResult: all realtime cases OK.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
