# Báo cáo Khắc phục Lỗi CORS & Redirect (HealthMate API)

Tài liệu này tổng hợp chi tiết về vấn đề CORS, tại sao nó xảy ra, cơ chế hoạt động của trình duyệt và cách chúng ta đã sửa lỗi để hệ thống hoạt động trơn tru.

---

## 1. Vấn đề 1: Lỗi "Redirect is not allowed for a preflight request"

### Nguyên nhân
Khi bạn dùng trình duyệt (hoặc công cụ test CORS) gọi đến `http://localhost:8080/medications`, trình duyệt thực hiện một bước gọi là **CORS Preflight**:
1. Trình duyệt gửi một yêu cầu `OPTIONS` trước để kiểm tra xem server có cho phép gọi API hay không.
2. Gin Framework (API Gateway) nhận được `/medications`. Theo mặc định, Gin sẽ tự động chuyển hướng (Redirect 301) về `/medications/` (thêm dấu gạch chéo).
3. **Quy tắc bảo mật**: Trình duyệt **không bao giờ** cho phép chuyển hướng (Redirection) đối với yêu cầu `OPTIONS`. Nếu server trả về 301/302 cho Preflight, trình duyệt sẽ chặn đứng yêu cầu ngay lập tức.

### Cách khắc phục
Chúng ta đã tắt tính năng tự động chuyển hướng của Gin trong `api-gateway/app/app.go`:
```go
router.RedirectTrailingSlash = false
router.RedirectFixedPath = false
```
Việc này đảm bảo server sẽ xử lý yêu cầu tại đúng URL được gọi mà không bắt trình duyệt phải chuyển hướng.

---

## 2. Vấn đề 2: Lỗi "404 Not Found" khi gọi URL không gạch chéo

### Nguyên nhân
Sau khi tắt tính năng chuyển hướng ở trên, nếu bạn gọi `/medications` (không gạch chéo), Gin sẽ không còn tự động tìm đến `/medications/` nữa. Nếu trong code chỉ khai báo route mẫu `/*proxyPath`, Gin sẽ coi `/medications` là không tồn tại (vì nó thiếu phần wildcard đằng sau).

### Cách khắc phục
Chúng ta khai báo **cả hai loại route** (Dual Routing) trong `api-gateway/app/router.go`:
```go
// Chấp nhận cả URL không gạch chéo
protected.Any("/medications", handlers.ReverseProxy(cfg.StorageHTTPURL, cfg.APIPrefix+"/medications"))
// VÀ chấp nhận cả URL có gạch chéo hoặc đường dẫn con
protected.Any("/medications/*proxyPath", handlers.ReverseProxy(cfg.StorageHTTPURL, cfg.APIPrefix+"/medications"))
```
Giải pháp này giúp hệ thống linh hoạt mà vẫn tuân thủ quy tắc không chuyển hướng của CORS.

---

## 3. Vấn đề 3: Origin 'null' từ file HTML cục bộ

### Nguyên nhân
Khi bạn mở file `test-cors.html` trực tiếp bằng trình duyệt (giao thức `file://`), trình duyệt sẽ gửi header `Origin: null` thay vì một domain cụ thể. 
Nhiều middleware CORS mặc định sẽ không chấp nhận giá trị `null`, dẫn đến lỗi "CORS policy: Response to preflight request doesn't pass access control check".

### Cách khắc phục
Cập nhật `api-gateway/internal/middleware/cors.go` để lấy đúng Origin từ header của yêu cầu và phản hồi lại chính Origin đó:
```go
origin := c.Request.Header.Get("Origin")
c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
```
Điều này cho phép file HTML cục bộ của bạn có thể giao tiếp với API một cách hợp lệ.

---

## 4. Tổng hợp các thay đổi Code

### `api-gateway/app/app.go`
- **Hành động**: Tắt tự động chuyển hướng.
- **Mục tiêu**: Ngăn chặn lỗi preflight redirect.

### `api-gateway/internal/middleware/cors.go`
- **Hành động**: Bổ sung hỗ trợ dynamic origin và các headers cần thiết (`Origin`, `Accept`, `Authorization`).
- **Mục tiêu**: Cho phép tester và các ứng dụng frontend (Flutter Web, v.v.) gọi API thành công.

### `api-gateway/app/router.go`
- **Hành động**: Đăng ký route kép cho tất cả các dịch vụ (Users, Groups, Metrics, Medications).
- **Mục tiêu**: Tránh lỗi 404 khi người dùng gọi URL không có dấu gạch chéo ở cuối.

---

## Kết luận
Hệ thống hiện tại đã có thể:
1. Xử lý preflight thành công mà không bị chuyển hướng.
2. Chấp nhận yêu cầu từ file HTML cục bộ (`Origin: null`).
3. Hoạt động đúng với cả URL có hoặc không có dấu gạch chéo ở cuối.

**Lưu ý**: Mỗi khi thay đổi code ở API Gateway, bạn cần chạy:
`docker compose up -d --build api-gateway`
để cập nhật container.
