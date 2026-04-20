import os

import cv2
import numpy as np


def crop_to_medication_body_bgr(img_bgr: np.ndarray) -> np.ndarray:
    """
    Ảnh dọc kiểu giấy đơn: bỏ phần đầu (logo, barcode, tiêu đề BV, thông tin BN),
    chỉ giữ thân có danh sách thuốc + SL + Cộng khoản. Không cắt đáy để không mất dòng cuối.
    Ảnh ngang/vuông (crop tay chỉ vùng thuốc) — giữ nguyên.
    """
    h, w = img_bgr.shape[:2]
    if h < 350:
        return img_bgr
    if h <= w * 1.08:
        return img_bgr
    try:
        top_frac = float(os.getenv("OCR_BODY_TOP_SKIP_FRAC", "0.18"))
    except ValueError:
        top_frac = 0.18
    top_frac = max(0.0, min(top_frac, 0.45))
    top = int(h * top_frac)
    if top >= h - 120:
        return img_bgr
    return img_bgr[top:, :]


def preprocess_image(raw: bytes, *, crop_body: bool = True) -> np.ndarray:
    arr = np.frombuffer(raw, np.uint8)
    img = cv2.imdecode(arr, cv2.IMREAD_COLOR)
    if img is None:
        raise ValueError("invalid image bytes")
    if crop_body:
        img = crop_to_medication_body_bgr(img)
    gray = cv2.cvtColor(img, cv2.COLOR_BGR2GRAY)
    gray = cv2.bilateralFilter(gray, 5, 35, 35)
    gray = cv2.convertScaleAbs(gray, alpha=1.2, beta=6)
    kernel = np.array([[0, -1, 0], [-1, 5, -1], [0, -1, 0]])
    sharp = cv2.filter2D(gray, -1, kernel)
    return sharp


def deskew(gray: np.ndarray) -> np.ndarray:
    # Estimate text angle from connected components and rotate back.
    th = cv2.threshold(gray, 0, 255, cv2.THRESH_BINARY + cv2.THRESH_OTSU)[1]
    inv = 255 - th
    coords = np.column_stack(np.where(inv > 0))
    if coords.size == 0:
        return gray
    angle = cv2.minAreaRect(coords)[-1]
    if angle < -45:
        angle = 90 + angle
    if abs(angle) < 0.5:
        return gray
    h, w = gray.shape[:2]
    m = cv2.getRotationMatrix2D((w // 2, h // 2), angle, 1.0)
    return cv2.warpAffine(gray, m, (w, h), flags=cv2.INTER_CUBIC, borderMode=cv2.BORDER_REPLICATE)


def ensure_min_ocr_size(img: np.ndarray, min_long_side: int = 1280) -> np.ndarray:
    """PaddleOCR dễ bỏ sót chữ nhỏ ở ảnh chụp xa; phóng to trước khi đọc."""
    h, w = img.shape[:2]
    long_side = max(h, w)
    target = min_long_side
    # Đơn dọc: cần độ phân giải dọc cao hơn để đọc hết các dòng thuốc phía dưới.
    if h > w * 1.22:
        target = max(target, 1680)
    if long_side >= target:
        return img
    scale = target / float(long_side)
    nw, nh = max(1, int(w * scale)), max(1, int(h * scale))
    return cv2.resize(img, (nw, nh), interpolation=cv2.INTER_CUBIC)
