import json
import urllib.error
import urllib.request
import uuid
from pathlib import Path

TOKEN = "eyJhbGciOiAiSFMyNTYiLCAidHlwIjogIkpXVCJ9.eyJzdWIiOiAiZGVtby11c2VyLW9jciIsICJ0eXBlIjogImFjY2VzcyIsICJlbWFpbCI6ICJkZW1vQGxvY2FsIiwgInJvbGUiOiAibWVtYmVyIiwgImV4cCI6IDE3NzU3NDE0OTh9.jBquiqiAkWfJsDHYi4qbS5EdgWh7DPO0NNnnxxvmDpI"
URL = "http://localhost:8080/ocr/prescriptions/parse"

PATHS = [
    r"C:\Users\aiiuu\.cursor\projects\c-Hoc-Tap-ATN\assets\c__Users_aiiuu_AppData_Roaming_Cursor_User_workspaceStorage_2f34aa181f92e02af3c554862384415e_images_665533573_2131369121039200_493664280806630949_n-ac6922a4-92dd-4d30-8abf-485e6b0079f3.png",
    r"C:\Users\aiiuu\.cursor\projects\c-Hoc-Tap-ATN\assets\c__Users_aiiuu_AppData_Roaming_Cursor_User_workspaceStorage_2f34aa181f92e02af3c554862384415e_images_662169494_948940104493450_1702267021556638530_n-62f65957-57d0-4b8f-a302-205a6800af52.png",
    r"C:\Users\aiiuu\.cursor\projects\c-Hoc-Tap-ATN\assets\c__Users_aiiuu_AppData_Roaming_Cursor_User_workspaceStorage_2f34aa181f92e02af3c554862384415e_images_662250004_2172952129911482_1433779290536495622_n-335bb6a5-0a52-4e17-b982-dcfbfc3d7d83.png",
    r"C:\Users\aiiuu\.cursor\projects\c-Hoc-Tap-ATN\assets\c__Users_aiiuu_AppData_Roaming_Cursor_User_workspaceStorage_2f34aa181f92e02af3c554862384415e_images_664022051_869372206115407_2643172541354722065_n-49c5565e-5233-4d05-b3bb-b970cc1fbe37.png",
    r"C:\Users\aiiuu\.cursor\projects\c-Hoc-Tap-ATN\assets\c__Users_aiiuu_AppData_Roaming_Cursor_User_workspaceStorage_2f34aa181f92e02af3c554862384415e_images_661072901_962564969521177_3641818443280458009_n-8e2e87b7-0194-4541-9daf-88ada9d7847d.png",
    r"C:\Users\aiiuu\.cursor\projects\c-Hoc-Tap-ATN\assets\c__Users_aiiuu_AppData_Roaming_Cursor_User_workspaceStorage_2f34aa181f92e02af3c554862384415e_images_642339050_1455056389447802_3264942585587069227_n-324313af-f5f8-41ec-acdd-9ac59560cb02.png",
    r"C:\Users\aiiuu\.cursor\projects\c-Hoc-Tap-ATN\assets\c__Users_aiiuu_AppData_Roaming_Cursor_User_workspaceStorage_2f34aa181f92e02af3c554862384415e_images_666239886_1593150598418746_8153015708731970082_n-b00f5386-c565-4075-bef5-ade17e507de8.png",
    r"C:\Users\aiiuu\.cursor\projects\c-Hoc-Tap-ATN\assets\c__Users_aiiuu_AppData_Roaming_Cursor_User_workspaceStorage_2f34aa181f92e02af3c554862384415e_images_667589965_972644985418269_9138318291086751492_n-e943ae5b-4493-421f-b458-fbda60260e1c.png",
    r"C:\Users\aiiuu\.cursor\projects\c-Hoc-Tap-ATN\assets\c__Users_aiiuu_AppData_Roaming_Cursor_User_workspaceStorage_2f34aa181f92e02af3c554862384415e_images_662618620_954224710670549_9132235593096998975_n-ee65feab-2919-40a8-82da-2d181dda808a.png",
    r"C:\Users\aiiuu\.cursor\projects\c-Hoc-Tap-ATN\assets\c__Users_aiiuu_AppData_Roaming_Cursor_User_workspaceStorage_2f34aa181f92e02af3c554862384415e_images_659813677_2416353085548154_7601677991925691977_n-1b5f4ca0-a416-41e1-82f2-f80fd65fb85c.png",
    r"C:\Users\aiiuu\.cursor\projects\c-Hoc-Tap-ATN\assets\c__Users_aiiuu_AppData_Roaming_Cursor_User_workspaceStorage_2f34aa181f92e02af3c554862384415e_images_662417215_2333657857143140_1860711517623749621_n-ba892d6c-5b93-47d4-afab-c5d46cc6a958.png",
    r"C:\Users\aiiuu\.cursor\projects\c-Hoc-Tap-ATN\assets\c__Users_aiiuu_AppData_Roaming_Cursor_User_workspaceStorage_2f34aa181f92e02af3c554862384415e_images_662271561_4856094791283499_1664676231828565317_n-8662f66b-2e6c-4668-b0bc-723e9d8c951f.png",
    r"C:\Users\aiiuu\.cursor\projects\c-Hoc-Tap-ATN\assets\c__Users_aiiuu_AppData_Roaming_Cursor_User_workspaceStorage_2f34aa181f92e02af3c554862384415e_images_666322264_1497815005076536_8018140414727040859_n-4a498dbb-98e9-4a2f-a391-56e9a71a2010.png",
    r"C:\Users\aiiuu\.cursor\projects\c-Hoc-Tap-ATN\assets\c__Users_aiiuu_AppData_Roaming_Cursor_User_workspaceStorage_2f34aa181f92e02af3c554862384415e_images_660337762_1950367022512887_6912860012122313822_n-c7727991-dca3-4070-9811-be3b4ced9525.png",
    r"C:\Users\aiiuu\.cursor\projects\c-Hoc-Tap-ATN\assets\c__Users_aiiuu_AppData_Roaming_Cursor_User_workspaceStorage_2f34aa181f92e02af3c554862384415e_images_665418263_913518704886739_6668465618317626612_n-a32c4663-2437-45e9-a57a-4c9e5da997ff.png",
]


def post_file(file_path: str) -> tuple[int, str]:
    boundary = "----Boundary" + uuid.uuid4().hex
    filename = Path(file_path).name
    with open(file_path, "rb") as f:
        data = f.read()

    body = (
        f"--{boundary}\r\n"
        f'Content-Disposition: form-data; name="file"; filename="{filename}"\r\n'
        f"Content-Type: image/png\r\n\r\n"
    ).encode() + data + f"\r\n--{boundary}--\r\n".encode()

    req = urllib.request.Request(URL, data=body, method="POST")
    req.add_header("Authorization", f"Bearer {TOKEN}")
    req.add_header("Content-Type", f"multipart/form-data; boundary={boundary}")
    req.add_header("Content-Length", str(len(body)))

    try:
        with urllib.request.urlopen(req, timeout=240) as resp:
            return resp.status, resp.read().decode("utf-8", "ignore")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", "ignore")


def main() -> None:
    results = []
    for idx, image_path in enumerate(PATHS, 1):
        status, text = post_file(image_path)
        item_count = -1
        raw_len = 0
        names = []
        error_preview = ""
        try:
            payload = json.loads(text)
            items = payload.get("items") or []
            raw_text = payload.get("raw_text") or ""
            item_count = len(items)
            raw_len = len(raw_text)
            names = [str(item.get("name", ""))[:120] for item in items[:5] if isinstance(item, dict)]
        except Exception:
            error_preview = text[:240]

        results.append(
            {
                "idx": idx,
                "file": Path(image_path).name,
                "status": status,
                "raw_len": raw_len,
                "items": item_count,
                "names": names,
                "error_preview": error_preview,
            }
        )
        print(f"[{idx:02d}] status={status} items={item_count} raw_len={raw_len} file={Path(image_path).name}")

    out_path = Path(r"c:\Hoc_Tap\ĐATN\HealthMate_BE\HealthMate\ocr-batch-result.json")
    out_path.write_text(json.dumps(results, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"\nSaved: {out_path}")


if __name__ == "__main__":
    main()
