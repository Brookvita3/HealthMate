#!/usr/bin/env python3
"""
Chạy sau khi:
1) docker compose (gateway + auth + storage + timescaledb) đã up
2) Đã nạp seed: scripts/e2e_sharing_permissions_seed.sql

Cài: pip install -r scripts/e2e_requirements.txt

Test quyền /metrics/charts, list medication shares, và auth API cap quyen thanh vien
(trùng Base / global trong sharing_permissions — JWT ký tay, khớp JWT_SECRET gateway).

Seed mở rộng: Eve (filter + chỉ HR/BP), Frank (full global), thêm calories_burned + blood_pressure;
spo2 không có base share -> mọi observer (trừ Alice xem chính mình) 403.
"""

from __future__ import annotations

import json
import os
import sys
import time
import uuid
from typing import Any

import jwt
import requests

ALICE = uuid.UUID("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
BOB = uuid.UUID("bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb")
CHARLIE = uuid.UUID("cccccccc-cccc-4ccc-8ccc-cccccccccccc")
DIANA = uuid.UUID("dddddddd-dddd-4ddd-8ddd-dddddddddddd")
EVE = uuid.UUID("eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee")
FRANK = uuid.UUID("ffffffff-ffff-4fff-8fff-ffffffffffff")
MED_ID = uuid.UUID("22222222-2222-4222-8222-222222222222")
GROUP_ID = uuid.UUID("11111111-1111-4111-8111-111111111111")


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


def charts(
    session: requests.Session,
    base: str,
    token: str,
    target_user: uuid.UUID,
    metric: str,
    range_: str = "7d",
) -> tuple[int, dict[str, Any] | list[Any] | None]:
    r = session.get(
        f"{base}/metrics/charts",
        params={
            "user_id": str(target_user),
            "metric_type": metric,
            "range": range_,
        },
        headers={"Authorization": f"Bearer {token}"},
        timeout=30,
    )
    body: dict[str, Any] | list[Any] | None
    try:
        body = r.json()
    except json.JSONDecodeError:
        body = None
    return r.status_code, body


def list_medication_shares(
    session: requests.Session,
    base: str,
    token: str,
    medication_id: uuid.UUID,
) -> tuple[int, Any]:
    r = session.get(
        f"{base}/medications/{medication_id}/shares",
        headers={"Authorization": f"Bearer {token}"},
        timeout=30,
    )
    try:
        return r.status_code, r.json()
    except json.JSONDecodeError:
        return r.status_code, r.text


def put_update_permissions(
    session: requests.Session,
    base: str,
    token: str,
    group_id: uuid.UUID,
    metric_types: list[str],
    target_user_id: uuid.UUID | None,
) -> tuple[int, Any]:
    body: dict[str, Any] = {"metric_types": metric_types}
    if target_user_id is not None:
        body["target_user_id"] = str(target_user_id)
    r = session.put(
        f"{base}/groups/{group_id}/permissions",
        json=body,
        headers={"Authorization": f"Bearer {token}"},
        timeout=30,
    )
    try:
        return r.status_code, r.json()
    except json.JSONDecodeError:
        return r.status_code, r.text


def post_set_permission(
    session: requests.Session,
    base: str,
    token: str,
    group_id: uuid.UUID,
    metric_type: str,
    enabled: bool,
    target_user_id: uuid.UUID | None,
) -> tuple[int, Any]:
    body: dict[str, Any] = {
        "metric_type": metric_type,
        "enabled": enabled,
    }
    if target_user_id is not None:
        body["target_user_id"] = str(target_user_id)
    r = session.post(
        f"{base}/groups/{group_id}/permissions",
        json=body,
        headers={"Authorization": f"Bearer {token}"},
        timeout=30,
    )
    try:
        return r.status_code, r.json()
    except json.JSONDecodeError:
        return r.status_code, r.text


def main() -> int:
    base = os.environ.get("E2E_GATEWAY_URL", "http://127.0.0.1:5002").rstrip("/")
    secret = os.environ.get("JWT_SECRET", "healthmate_jwt_secret_dev").strip('"')

    session = requests.Session()

    tokens: dict[str, str] = {
        "alice": mint_token(ALICE, "e2e_share_alice@healthmate.test", secret),
        "bob": mint_token(BOB, "e2e_share_bob@healthmate.test", secret),
        "charlie": mint_token(CHARLIE, "e2e_share_charlie@healthmate.test", secret),
        "diana": mint_token(DIANA, "e2e_share_diana@healthmate.test", secret),
        "eve": mint_token(EVE, "e2e_share_eve@healthmate.test", secret),
        "frank": mint_token(FRANK, "e2e_share_frank@healthmate.test", secret),
    }

    # (title, who, target, metric, want_http, note)
    cases: list[tuple[str, str, uuid.UUID, str, int, str]] = [
        # Co ban + nhieu metric
        ("Bob -> Alice heart_rate", "bob", ALICE, "heart_rate", 200, "global, no filter"),
        ("Bob -> Alice steps_count", "bob", ALICE, "steps_count", 200, "global"),
        ("Bob -> Alice calories_burned", "bob", ALICE, "calories_burned", 200, "global"),
        ("Bob -> Alice blood_pressure", "bob", ALICE, "blood_pressure", 200, "global"),
        ("Bob -> Alice spo2 (no base share)", "bob", ALICE, "spo2", 403, "no sharing_permissions row"),
        # Charlie: filter + chi steps cu the
        ("Charlie -> Alice heart_rate", "charlie", ALICE, "heart_rate", 403, "filter, no HR spec"),
        ("Charlie -> Alice steps_count", "charlie", ALICE, "steps_count", 200, "specific grant"),
        ("Charlie -> Alice calories_burned", "charlie", ALICE, "calories_burned", 403, "filter blocks global"),
        ("Charlie -> Alice blood_pressure", "charlie", ALICE, "blood_pressure", 403, "filter blocks global"),
        ("Charlie -> Alice spo2", "charlie", ALICE, "spo2", 403, "no base"),
        # Ngoai nhom
        ("Diana -> Alice heart_rate", "diana", ALICE, "heart_rate", 403, "not in shared group"),
        # Chu so lieu xem chinh minh
        ("Alice -> Alice heart_rate", "alice", ALICE, "heart_rate", 200, "self always ok"),
        ("Alice -> Alice spo2", "alice", ALICE, "spo2", 200, "self; data may be empty but 200"),
        # Frank: member khong filter -> tat ca base
        ("Frank -> Alice heart_rate", "frank", ALICE, "heart_rate", 200, "full global"),
        ("Frank -> Alice calories_burned", "frank", ALICE, "calories_burned", 200, "full global"),
        ("Frank -> Alice blood_pressure", "frank", ALICE, "blood_pressure", 200, "full global"),
        ("Frank -> Alice steps_count", "frank", ALICE, "steps_count", 200, "full global"),
        ("Frank -> Alice spo2", "frank", ALICE, "spo2", 403, "no base spo2"),
        # Eve: filter + chi HR/BP cu the
        ("Eve -> Alice heart_rate", "eve", ALICE, "heart_rate", 200, "specific HR"),
        ("Eve -> Alice blood_pressure", "eve", ALICE, "blood_pressure", 200, "specific BP"),
        ("Eve -> Alice steps_count", "eve", ALICE, "steps_count", 403, "filter, no steps spec"),
        ("Eve -> Alice calories_burned", "eve", ALICE, "calories_burned", 403, "filter, no cal spec"),
        ("Eve -> Alice spo2", "eve", ALICE, "spo2", 403, "no base spo2"),
        # Metric khong ton tai trong metric_types -> sau CheckAccess van 403 neu khong co base
        ("Bob -> Alice weight (unknown)", "bob", ALICE, "weight", 403, "no share row for weight"),
        # FE sai ten calo: khong co base sharing_permissions cho calories_burnt
        ("Bob -> Alice calories_burnt typo", "bob", ALICE, "calories_burnt", 403, "no base row for metric name"),
        # Chu xem chinh minh nhung metric khong co trong DB schema metric_types
        ("Alice -> Alice weight unknown type", "alice", ALICE, "weight", 400, "unknown metric type after self bypass"),
    ]

    failed = 0
    print(f"Gateway: {base}\n")

    for title, who, target, metric, want, note in cases:
        token = tokens[who]
        code, body = charts(session, base, token, target, metric)
        ok = code == want
        if not ok:
            failed += 1
        status = "PASS" if ok else "FAIL"
        detail = ""
        if isinstance(body, dict) and "data" in body:
            pts = body.get("data")
            detail = f" points={len(pts) if isinstance(pts, list) else '?'}"
        elif isinstance(body, dict) and "error" in body:
            detail = f" error={body.get('error')!r}"
        print(f"[{status}] {title} -> HTTP {code} (expect {want}) - {note}{detail}")

    # Medication shares: chi chu thuoc duoc list; Bob/Charlie/Eve/Frank deu khong
    med_checks: list[tuple[str, str, int, str]] = [
        ("Alice list medication shares", "alice", 200, "owner"),
        ("Bob list medication shares", "bob", 403, "not owner"),
        ("Charlie list medication shares", "charlie", 403, "not owner"),
        ("Eve list medication shares", "eve", 403, "not owner"),
    ]
    for title, who, want, note in med_checks:
        code, body = list_medication_shares(session, base, tokens[who], MED_ID)
        ok = code == want
        if want == 200:
            ok = ok and isinstance(body, list)
        if not ok:
            failed += 1
        print(
            f"[{'PASS' if ok else 'FAIL'}] {title} -> {code} (expect {want}) - {note}"
        )

    # Auth-service: quyen rieng cho thanh vien phai la con cua Base (global NULL) cua chu so lieu.
    # Seed khong co base spo2 -> khong duoc gan spo2 rieng cho Bob/Frank qua API.
    tok_alice = tokens["alice"]
    hierarchy_cases: list[tuple[str, tuple[int, Any], int, str]] = [
        (
            "PUT permissions Bob: HR+spo2 (spo2 ngoai Base)",
            put_update_permissions(
                session,
                base,
                tok_alice,
                GROUP_ID,
                ["heart_rate", "spo2"],
                BOB,
            ),
            400,
            "UpdateSharing rejects metric not in global base",
        ),
        (
            "PUT permissions Frank: chi spo2 (ngoai Base)",
            put_update_permissions(
                session,
                base,
                tok_alice,
                GROUP_ID,
                ["spo2"],
                FRANK,
            ),
            400,
            "subset-of-base check",
        ),
        (
            "POST enable spo2 rieng cho Bob (khong co global spo2)",
            post_set_permission(
                session,
                base,
                tok_alice,
                GROUP_ID,
                "spo2",
                True,
                BOB,
            ),
            403,
            "EnableSharing requires metric in global first",
        ),
    ]
    print("")
    for title, (code, body), want, note in hierarchy_cases:
        ok = code == want
        if not ok:
            failed += 1
        detail = ""
        if isinstance(body, dict):
            detail = f" body={body!r}"
        print(
            f"[{'PASS' if ok else 'FAIL'}] {title} -> HTTP {code} (expect {want}) - {note}{detail}"
        )

    print(f"\nDone. Failed: {failed}")
    return 1 if failed else 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except requests.exceptions.ConnectionError as e:
        print(
            "Khong ket noi gateway. Chay docker compose va mo port 5002.",
            file=sys.stderr,
        )
        print(e, file=sys.stderr)
        raise SystemExit(2)
