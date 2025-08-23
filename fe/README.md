# Flutter Project

## Giới thiệu

## Yêu cầu hệ thống

- Flutter SDK: Phiên bản 3.0 trở lên.
- Dart SDK: Phiên bản 2.17 trở lên.
- Công cụ phát triển: Android Studio, Visual Studio Code, hoặc Xcode (cho iOS).
- Hệ điều hành: Windows, macOS, hoặc Linux.

## Cài đặt

1. **Clone repository**:
   ```
   git clone <URL của repository>
   cd <tên thư mục dự án>
   ```

2. **Cài đặt dependencies**:
   Chạy lệnh sau để tải về các gói phụ thuộc:
   ```
   flutter pub get
   ```

3. **Cấu hình môi trường**:
   - Đảm bảo bạn đã thiết lập Flutter đúng cách bằng lệnh `flutter doctor`.
   - Nếu phát triển cho iOS, cần có Xcode và CocoaPods.

## Chạy ứng dụng

- **Chạy trên emulator/simulator**:
  ```
  flutter run
  ```

- **Build APK cho Android**:
  ```
  flutter build apk
  ```

- **Build IPA cho iOS**:
  ```
  flutter build ios
  ```

## Cấu trúc thư mục

Dự án được tổ chức theo kiến trúc phân lớp để dễ bảo trì và mở rộng:

- **lib/**: Thư mục chính chứa mã nguồn Dart.
  - **core/**: Các thành phần cốt lõi chung.
    - **constants/**: Các hằng số cố định (ví dụ: API keys, strings).
    - **extensions/**: Các extension mở rộng cho các lớp cơ bản của Dart/Flutter.
    - **theme/**: Cấu hình theme (màu sắc, font chữ) cho ứng dụng.
    - **utils/**: Các hàm tiện ích chung (ví dụ: helper functions, validators).
  - **data/**: Lớp dữ liệu, xử lý nguồn dữ liệu.
    - **enums/**: Các enum định nghĩa (ví dụ: trạng thái, loại dữ liệu).
    - **models/**: Các model dữ liệu (DTO, entities).
    - **repositories/**: Các repository để truy cập dữ liệu (local/remote).
    - **services/**: Các service kết nối API, database, hoặc third-party.
  - **presentation/**: Lớp giao diện người dùng.
    - **present/**: Các widget, screen, và bloc/state management.
  - **main.dart**: File entry point của ứng dụng, khởi tạo app và route.

## Công nghệ sử dụng

- **Flutter**: Framework chính cho UI.
- **Dart**: Ngôn ngữ lập trình.
- **State Management**: Bloc.
- **Dependencies**: Các gói như `http` cho API, `shared_preferences` cho lưu trữ local (xem `pubspec.yaml` để biết chi tiết).

## Hướng dẫn phát triển

- **Thêm feature mới**: Tạo thư mục con trong `presentation/` cho UI, và tương ứng trong `data/` cho logic dữ liệu.
- **Test**: Chạy unit test bằng `flutter test`.
- **Linting**: Sử dụng `flutter analyze` để kiểm tra code style.