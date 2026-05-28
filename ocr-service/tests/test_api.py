import os
import sys
from pathlib import Path
import pytest
from unittest.mock import MagicMock

# We want to check if we can import the app. If we can't (due to missing fastapi,
# python-multipart, etc. in lightweight test environments),
# we skip this entire test module so it doesn't break lightweight/CI parser tests.
try:
    # Helper to conditionally mock missing dependencies so we don't overwrite
    # real packages that pytest uses (like numpy) if they are present.
    def mock_if_missing(module_name):
        try:
            __import__(module_name)
        except ImportError:
            if module_name == 'numpy':
                class DummyBool:
                    pass
                class DummyNdarray:
                    pass
                mock_np = MagicMock()
                mock_np.bool_ = DummyBool
                mock_np.ndarray = DummyNdarray
                sys.modules[module_name] = mock_np
            else:
                sys.modules[module_name] = MagicMock()

    mock_if_missing('cv2')
    mock_if_missing('numpy')
    mock_if_missing('paddleocr')
    mock_if_missing('paddlepaddle')
    mock_if_missing('prometheus_fastapi_instrumentator')

    # Make root importable
    sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

    from app.main import app
    from fastapi.testclient import TestClient
    client = TestClient(app)
except Exception as e:
    pytestmark = pytest.mark.skip(reason=f"Skipping API tests because of missing dependencies: {e}")
    client = None


def test_health_ready():
    if client is None:
        pytest.skip("TestClient not initialized")
    r1 = client.get("/health")
    assert r1.status_code == 200
    assert r1.json() == {"status": "ok"}

    r2 = client.get("/ready")
    assert r2.status_code == 200
    assert r2.json() == {"status": "UP"}


def test_parse_prescription_empty_file():
    if client is None:
        pytest.skip("TestClient not initialized")
    # Sending empty file
    files = {"file": ("empty.png", b"", "image/png")}
    response = client.post("/ocr/prescriptions/parse", files=files)
    assert response.status_code == 400
    assert "empty file" in response.json()["detail"]


def test_parse_prescription_invalid_format():
    if client is None:
        pytest.skip("TestClient not initialized")
    # Sending invalid text file
    files = {"file": ("dummy.txt", b"not_an_image_header", "text/plain")}
    response = client.post("/ocr/prescriptions/parse", files=files)
    assert response.status_code == 415
    assert "Chỉ chấp nhận" in response.json()["detail"]


def test_parse_prescription_too_large():
    if client is None:
        pytest.skip("TestClient not initialized")
    # Send large dummy file starting with correct magic bytes
    files = {"file": ("large.png", b"\x89PNG" + b"\x00" * (20 * 1024 * 1024 + 1), "image/png")}
    response = client.post("/ocr/prescriptions/parse", files=files)
    assert response.status_code == 413
    assert "File quá lớn" in response.json()["detail"]


@pytest.mark.filterwarnings("ignore:coroutine 'patch' was never awaited")
def test_parse_prescription_temp_file_lifecycle():
    if client is None:
        pytest.skip("TestClient not initialized")
    from unittest.mock import patch
    
    with patch("app.main._process_image_sync") as mock_process:
        # Mocking CPU intensive processing
        mock_process.return_value = {
            "raw_text": "metformin",
            "items": [],
            "warnings": [],
            "meta": {"pass_used": "mock"}
        }

        # Use valid small PNG magic bytes
        files = {"file": ("test.png", b"\x89PNG_dummy_bytes", "image/png")}
        
        # We want to capture the path of the temporary file passed to _process_image_sync
        # to verify that it existed during execution and is deleted afterwards.
        temp_file_paths = []
        
        def side_effect(path):
            assert os.path.exists(path)
            temp_file_paths.append(path)
            return mock_process.return_value

        mock_process.side_effect = side_effect

        response = client.post("/ocr/prescriptions/parse", files=files)
        assert response.status_code == 200
        
        # Assert _process_image_sync was called
        assert len(temp_file_paths) == 1
        # Verify the temporary file has been deleted from disk
        assert not os.path.exists(temp_file_paths[0])
