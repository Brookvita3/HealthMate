"""
paddle_cl.py
-----------
A class-based interface for PaddleOCR-VL using llama-cpp-python.
Supports both image file paths and in-memory numpy image arrays.
"""

from __future__ import annotations

import base64
import io
import os
# import random
from typing import List, Optional, Union

import numpy as np
import ctypes
from llama_cpp import llama_log_set, Llama
from llama_cpp.llama_chat_format import PaddleOCRChatHandler
from PIL import Image


# ---------------------------------------------------------------------------
# Set random seeds for reproducibility
# ---------------------------------------------------------------------------
# def set_random_seed(seed: int = 10) -> None:
#     """Set seeds for Python, NumPy, and llama.cpp for maximum reproducibility."""
#     random.seed(seed)
#     np.random.seed(seed)
    
#     try:
#         # Try to set seed for llama.cpp (affects sampling)
#         from llama_cpp import llama_set_rng_seed
#         llama_set_rng_seed(seed)
#         print(f"Random seed set to {seed}")
#     except Exception:
#         print(f"Random seed set to {seed}")
#         pass


# ---------------------------------------------------------------------------
# MIME type lookup based on file extension
# ---------------------------------------------------------------------------
_IMAGE_MIME_TYPES: dict[str, str] = {
    ".png": "image/png",
    ".jpg": "image/jpeg",
    ".jpeg": "image/jpeg",
    ".gif": "image/gif",
    ".webp": "image/webp",
    ".avif": "image/avif",
    ".jp2": "image/jp2",
    ".j2k": "image/jp2",
    ".jpx": "image/jp2",
    ".bmp": "image/bmp",
    ".tif": "image/tiff",
    ".tiff": "image/tiff",
    ".dds": "image/vnd-ms.dds",
    ".pbm": "image/x-portable-bitmap",
    ".pgm": "image/x-portable-graymap",
    ".ppm": "image/x-portable-pixmap",
    # Add more as needed
}

# ---------------------------------------------------------------------------
# Pre-defined task prompts
# ---------------------------------------------------------------------------
TASK_PROMPTS: dict[str, str] = {
    "ocr": "OCR:",
    "table": "Table Recognition:",
    "formula": "Formula Recognition:",
    "chart": "Chart Recognition:",
    "spotting": "Spotting:",
    "seal": "Seal Recognition:",
}


# === Suppress all llama.cpp logs ===
def silent_log_callback(level: int, message: bytes, user_data: ctypes.c_void_p):
    pass


log_callback = ctypes.CFUNCTYPE(None, ctypes.c_int, ctypes.c_char_p, ctypes.c_void_p)(silent_log_callback)
llama_log_set(log_callback, ctypes.c_void_p())


class PaddleOCRVision:
    """High-level interface for PaddleOCR-VL multimodal model."""

    def __init__(
        self,
        model_path: str,
        mmproj_path: str,
        n_gpu_layers: int = 999,
        n_ctx: int = 0,
        n_batch: int = 2048,
        verbose: int = 0,
        # seed: Optional[int] = 100,
        **llama_kwargs,
    ) -> None:
    #     if seed is not None:
    #         set_random_seed(seed)

        self.llm = Llama(
            model_path=model_path,
            chat_handler=PaddleOCRChatHandler(clip_model_path=mmproj_path),
            n_gpu_layers=n_gpu_layers,
            num_predict=-1,
            n_ctx=n_ctx,
            n_batch=n_batch,
            verbose=verbose,
            **llama_kwargs,
        )

    # ------------------------------------------------------------------
    # Image conversion helpers
    # ------------------------------------------------------------------
    @staticmethod
    def _image_path_to_data_uri(
        file_path: str,
        fallback_mime: str = "application/octet-stream",
    ) -> str:
        if not os.path.isfile(file_path):
            raise FileNotFoundError(f"Image file not found: {file_path}")

        ext = os.path.splitext(file_path)[1].lower()
        mime = _IMAGE_MIME_TYPES.get(ext, fallback_mime)

        with open(file_path, "rb") as f:
            encoded = base64.b64encode(f.read()).decode("utf-8")
        return f"data:{mime};base64,{encoded}"

    @staticmethod
    def _array_to_data_uri(
        array: np.ndarray,
        image_format: str = "PNG",
    ) -> str:
        if array.dtype != np.uint8:
            array = np.clip(array, 0, 255).astype(np.uint8)

        if array.ndim == 2:
            pil_image = Image.fromarray(array, mode="L").convert("RGB")
        elif array.ndim == 3 and array.shape[2] == 3:
            pil_image = Image.fromarray(array, mode="RGB")
        elif array.ndim == 3 and array.shape[2] == 4:
            pil_image = Image.fromarray(array, mode="RGBA").convert("RGB")
        else:
            raise ValueError(f"Unsupported array shape {array.shape}")

        buffer = io.BytesIO()
        pil_image.save(buffer, format=image_format)
        encoded = base64.b64encode(buffer.getvalue()).decode("utf-8")

        ext = f".{image_format.lower()}"
        mime = _IMAGE_MIME_TYPES.get(ext, "application/octet-stream")
        return f"data:{mime};base64,{encoded}"

    # ------------------------------------------------------------------
    # Main OCR method
    # ------------------------------------------------------------------
    def ocr(
        self,
        images: Union[str, np.ndarray, List[Union[str, np.ndarray]]],
        prompt: Optional[str] = None,
        task: Optional[str] = None,
        max_tokens: int = 2048,
        array_format: str = "PNG",
        temperature: float = 0.0,
        top_p: float = 1.0,
        seed: Optional[int] = None,
        **completion_kwargs,
    ) -> str:
        """Run vision-language inference."""

        # Resolve prompt
        if task is not None:
            if task not in TASK_PROMPTS:
                raise ValueError(f"Unknown task '{task}'. Available: {list(TASK_PROMPTS.keys())}")
            prompt = TASK_PROMPTS[task]
        elif prompt is None:
            prompt = TASK_PROMPTS["ocr"]

        # Normalize images to list
        if not isinstance(images, list):
            images = [images]

        # Build message content
        user_content: list[dict] = []
        for img in images:
            if isinstance(img, str):
                data_uri = self._image_path_to_data_uri(img)
            elif isinstance(img, np.ndarray):
                data_uri = self._array_to_data_uri(img, image_format=array_format)
            else:
                raise TypeError(f"Unsupported image type: {type(img)}")

            user_content.append({"type": "image_url", "image_url": {"url": data_uri}})

        user_content.append({"type": "text", "text": prompt})

        # Run inference with deterministic settings
        response = self.llm.create_chat_completion(
            messages=[{"role": "user", "content": user_content}],
            max_tokens=max_tokens,
            temperature=temperature,
            top_p=top_p,
            seed=seed,
            **completion_kwargs,
        )
        return response["choices"][0]["message"]["content"]


# ---------------------------------------------------------------------------
# Example usage
# ---------------------------------------------------------------------------
if __name__ == "__main__":
    MODEL = r"ckpts/paddle_gguf/PaddleOCR-VL-1.5-Q8_0.gguf"
    MMPROJ = r"ckpts/paddle_gguf/mmproj-BF16.gguf"

    # Initialize with seed
    ocr_engine = PaddleOCRVision(
        model_path=MODEL,
        mmproj_path=MMPROJ,
        n_gpu_layers=999,
        n_ctx=0,
        n_batch=2048,
        verbose=0,
        # seed=100,                    # Set seed at initialization
    )

    import cv2
    import time

    # Example with numpy array
    img_path = "scripts/dataset/images/z7748543963758_9cd98091e2edf30c541ce929f4d7ae70.jpg"
    img_bgr = cv2.imread(img_path)

    if img_bgr is not None:
        img_rgb = cv2.cvtColor(img_bgr, cv2.COLOR_BGR2RGB)
        
        t1 = time.time()
        result = ocr_engine.ocr(
            images=img_rgb,
            task="ocr",
            temperature=0.3,      # Best for deterministic output
            top_p=1.0,
            seed=100
        )
        print("Output:\n", result)
        print(f"Time: {time.time() - t1:.2f} seconds")