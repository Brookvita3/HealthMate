### Lộ Trình Học Tập: Từ Zero đến Hero với Kubernetes và Microservices

**Phần 1: Nền Tảng Vững Chắc (Cơ bản)**

- **Bài 1: Nhập môn Kiến trúc Microservice.**
  - Kiến trúc Monolithic là gì? Ưu và nhược điểm.
  - Microservice là gì? Tại sao nó ra đời?
  - So sánh trực quan Monolithic và Microservice.
  - Các nguyên tắc cốt lõi của kiến trúc Microservice.
- **Bài 2: Giới thiệu về Container và Docker.**
  - Container là gì? So sánh với máy ảo (VM).
  - Docker là gì? Các khái niệm cơ bản (Image, Container, Dockerfile, Docker Hub).
  - Thực hành: "Docker hóa" một ứng dụng đơn giản.
- **Bài 3: Kubernetes - Người Nhạc Trưởng Cho Dàn Nhạc Microservices.**
  - Tại sao cần Kubernetes khi đã có Docker?
  - Kubernetes là gì? Các tính năng chính.
  - Kiến trúc tổng quan của Kubernetes (Control Plane và Worker Nodes).
  - Các khái niệm cốt lõi: Pod, Node, Service, Deployment, ReplicaSet.

**Phần 2: Chinh Phục Kubernetes (Trung cấp)**

- **Bài 4: Thực Hành Triển Khai Ứng Dụng Microservice Đầu Tiên.**
  - Thiết lập môi trường Kubernetes local (sử dụng Minikube hoặc Kind).
  - Viết file manifest YAML cho ứng dụng (Frontend, Backend, Database).
  - Sử dụng `kubectl` để triển khai và quản lý ứng dụng.
  - Tìm hiểu về Service Discovery và Network trong K8s.
- **Bài 5: Quản Lý Cấu Hình và Dữ Liệu.**
  - ConfigMap và Secret: Quản lý cấu hình ứng dụng một cách linh hoạt và an toàn.
  - Volumes, PersistentVolumes (PV) và PersistentVolumeClaims (PVC): Quản lý dữ liệu có trạng thái cho database.
- **Bài 6: Health Checks và Vòng Đời Của Pod.**
  - Liveness, Readiness và Startup Probes: Đảm bảo ứng dụng luôn khỏe mạnh.
  - Vòng đời của một Pod và cách K8s quản lý chúng.

**Phần 3: Xây Dựng Hệ Thống Chuyên Nghiệp (Nâng cao)**

- **Bài 7: Ingress và Quản Lý API Gateway.**
  - Ingress Controller: Đưa traffic từ bên ngoài vào trong cluster.
  - API Gateway: Vai trò và cách triển khai trong kiến trúc Microservice.
- **Bài 8: DevOps và CI/CD với Kubernetes.**
  - Helm: Trình quản lý gói cho Kubernetes.
  - Xây dựng pipeline CI/CD tự động build Docker image và deploy lên K8s (sử dụng Jenkins, GitLab CI hoặc GitHub Actions).
- **Bài 9: Giám Sát, Ghi Log và Bảo Mật.**
  - Monitoring với Prometheus và Grafana.
  - Centralized Logging với EFK Stack (Elasticsearch, Fluentd, Kibana).
  - Các khái niệm bảo mật cơ bản trong K8s: RBAC, Network Policies.































































---

## Authorization Code Flow và ID Token Flow

Khi tích hợp đăng nhập Google vào ứng dụng, có hai luồng (flow) chính thường được sử dụng. Việc lựa chọn luồng nào phụ thuộc vào kiến trúc ứng dụng của bạn (Web server, Single Page App, Mobile App) và mục đích sử dụng (chỉ xác thực hay cần ủy quyền truy cập tài nguyên).

### 📦 Luồng 1: Authorization Code Flow (Có Redirect)

Đây là luồng OAuth 2.0 đầy đủ và an toàn nhất, thường được sử dụng cho các ứng dụng web có backend (server-side). Luồng này không chỉ giúp **xác thực** (Authentication - bạn là ai?) mà còn thực hiện việc **ủy quyền** (Authorization - bạn được phép làm gì?).

#### Các bước thực hiện:

**1. FE/BE → Google: Yêu cầu Authorization Code**

Ứng dụng của bạn (FE hoặc BE) sẽ điều hướng người dùng sang trang đăng nhập của Google.

- **Endpoint của Google:** `https://accounts.google.com/o/oauth2/v2/auth`
- **Các tham số quan trọng gửi kèm:**
  - `client_id`: ID định danh ứng dụng của bạn đã đăng ký với Google.
  - `redirect_uri`: URL mà Google sẽ gọi lại sau khi người dùng đăng nhập thành công. URL này phải được đăng ký trước trong Google Cloud Console để đảm bảo an toàn.
  - `response_type=code`: Chỉ định rằng bạn muốn nhận về một `authorization_code`.
  - `scope`: Xác định các quyền mà ứng dụng của bạn muốn truy cập (ví dụ: `email`, `profile`, `https://www.googleapis.com/auth/calendar`).
  - `state`: Một chuỗi ngẫu nhiên do ứng dụng của bạn tạo ra để chống lại tấn công CSRF (Cross-Site Request Forgery). Giá trị này sẽ được Google gửi lại và bạn cần kiểm tra tính toàn vẹn của nó.

**Ví dụ URL đầy đủ:**

```
https://accounts.google.com/o/oauth2/v2/auth?client_id=xxx.apps.googleusercontent.com&redirect_uri=http://localhost:8080/callback&response_type=code&scope=email%20profile&state=xyz
```

**2. Google → Backend: Trả về Authorization Code**

Sau khi người dùng đăng nhập và chấp thuận các quyền, Google sẽ điều hướng (redirect) người dùng trở lại `redirect_uri` của bạn, đính kèm `authorization_code` và `state`.

**Ví dụ URL callback:**

```
http://localhost:8080/callback?code=abc123&state=xyz
```

Lúc này, backend của bạn cần kiểm tra xem giá trị `state` nhận về có khớp với giá trị đã gửi đi ở bước 1 không.

**3. Backend → Google: Đổi Code Lấy Token**

Backend của bạn gửi một yêu cầu POST đến `token_uri` của Google để đổi `authorization_code` lấy các token cần thiết.

- Yêu cầu này phải chứa: `code`, `client_id`, `client_secret`, `redirect_uri`, và `grant_type=authorization_code`.
- `client_secret` là một khóa bí mật chỉ backend biết, giúp xác thực rằng yêu cầu này thực sự đến từ ứng dụng của bạn.

**4. Google → Backend: Trả về Tokens**

Nếu `code` hợp lệ, Google sẽ trả về một JSON object chứa các loại token:

```json
{
  "access_token": "ya29.a0...",
  "refresh_token": "1//0...",
  "expires_in": 3600,
  "id_token": "eyJhbGciOiJSUzI1NiIs..."
}
```

- `access_token`: Dùng để gọi các Google API mà người dùng đã cấp quyền (phạm vi được định nghĩa trong `scope`).
- `refresh_token`: Dùng để lấy `access_token` mới khi nó hết hạn mà không cần người dùng đăng nhập lại (chỉ được cấp trong lần đầu tiên với `access_type=offline`).
- `id_token`: Một chuỗi JWT chứa thông tin định danh của người dùng (như email, tên, avatar).

**5. Backend: Xử lý và Phát hành Token Nội bộ**

- **Sử dụng token:** Backend có thể dùng `access_token` để truy cập tài nguyên Google (ví dụ: đọc lịch Google Calendar). Hoặc chỉ cần giải mã `id_token` để lấy thông tin người dùng và xác thực họ.
- **Phát hành token nội bộ:** Thông thường, backend sẽ không trả `access_token` của Google về cho frontend. Thay vào đó, nó sẽ tạo ra một bộ token nội bộ (thường là JWT) để quản lý phiên đăng nhập của người dùng trên ứng dụng của bạn.

---

### 📦 Luồng 2: ID Token Flow (Luồng bạn đang dùng)

Luồng này đơn giản hơn, thường được dùng cho các ứng dụng phía client như Single Page Apps (SPA) hoặc ứng dụng di động. Mục đích chính của luồng này là **xác thực** người dùng một cách nhanh chóng.

#### Các bước thực hiện:

**1. FE → Google: Đăng nhập và nhận ID Token**

- Frontend sử dụng thư viện của Google (ví dụ: Google Identity Services) để hiển thị nút "Sign in with Google".
- Khi người dùng đăng nhập thành công, thư viện sẽ trả về trực tiếp `id_token` cho frontend, không cần qua bước redirect và lấy `authorization_code`.

**2. FE → Backend: Gửi ID Token**

Frontend gửi `id_token` này lên backend thông qua một request (thường là POST) qua HTTPS.

**3. Backend: Xác thực (Verify) ID Token**

Đây là bước cực kỳ quan trọng. Backend phải xác thực `id_token` để đảm bảo nó hợp lệ và thực sự do Google cấp. Các bước kiểm tra bao gồm:

- Kiểm tra chữ ký của token bằng public key của Google.
- Kiểm tra `aud` (audience) claim phải khớp với `client_id` của ứng dụng bạn.
- Kiểm tra `iss` (issuer) claim phải là `accounts.google.com` hoặc `https://accounts.google.com`.
- Kiểm tra token chưa hết hạn (dựa vào `exp` claim).

Cách tốt nhất là sử dụng thư viện chính thức của Google để thực hiện việc này.

**4. Backend: Phát hành Token Nội bộ**

Sau khi xác thực thành công và lấy được thông tin người dùng từ `id_token` (như email, sub-id), backend sẽ tạo ra token nội bộ (JWT) và gửi về cho frontend để duy trì phiên đăng nhập.

### 👉 Tổng kết: Nên dùng luồng nào?

- **Authorization Code Flow:** Là luồng đầy đủ chức năng, phù hợp khi bạn cần **ủy quyền** cho backend truy cập vào dữ liệu của người dùng trên Google (ví dụ: Google Drive, Calendar, Photos). An toàn hơn vì các token quan trọng được trao đổi trực tiếp giữa server-to-server.
- **ID Token Flow:** Là luồng gọn nhẹ, tập trung vào việc **xác thực** danh tính người dùng. Rất phù hợp cho các ứng dụng chỉ cần chức năng "Đăng nhập bằng Google" mà không cần gọi thêm các API khác của Google từ phía backend.

Chính xác, đây là một câu hỏi cực kỳ quan trọng và là một khái niệm cốt lõi trong lập trình Go hiện đại, đặc biệt là với các ứng dụng mạng như của bạn.

Hãy giải thích một cách đơn giản nhất.

### `context` là gì?

Hãy tưởng tượng mỗi một request từ người dùng vào server của bạn là một **"nhiệm vụ"** (task). Nhiệm vụ này có thể phải đi qua nhiều bước: từ `Handler` → `Service` → `Repository` → gọi đến Database hoặc một API khác.

`context.Context` (thường được viết tắt là `ctx`) chính là **"chiếc cặp tài liệu" đi kèm với nhiệm vụ đó**. Chiếc cặp này mang theo thông tin về vòng đời và phạm vi của chính nhiệm vụ đó.

Nó giúp trả lời các câu hỏi sau:

1.  **"Nhiệm vụ này còn cần thực hiện nữa không?" (Cancellation Signal - Tín hiệu hủy bỏ)**

    - **Tình huống:** Người dùng gửi một request để lấy dữ liệu (mất 5 giây để xử lý). Nhưng sau 1 giây, họ đóng tab trình duyệt.
    - **Không có `context`:** Server của bạn không biết điều này và vẫn tiếp tục xử lý request trong 4 giây còn lại, lãng phí tài nguyên (CPU, kết nối DB).
    - **Có `context`:** Khi người dùng đóng tab, web framework sẽ "hủy" (cancel) chiếc cặp `context` này. Ở bất kỳ bước nào trong chuỗi xử lý (ví dụ đang query DB), code của bạn có thể kiểm tra `if ctx.Done()` và dừng công việc ngay lập tức. Điều này giúp giải phóng tài nguyên hệ thống.

2.  **"Nhiệm vụ này có thời hạn bao lâu?" (Deadlines/Timeouts - Hạn chót)**

    - **Tình huống:** Trong chuỗi xử lý, bạn cần gọi một API của bên thứ ba. Bạn không muốn ứng dụng của mình bị "treo" vô thời hạn nếu API đó bị chậm hoặc không phản hồi.
    - **Cách làm:** Bạn có thể tạo một `context` con với thời hạn, ví dụ: `ctx, cancel := context.WithTimeout(parentCtx, 500*time.Millisecond)`. Sau đó bạn dùng `ctx` này để gọi API. Nếu quá 500ms mà API chưa trả về, `context` sẽ tự động bị hủy, và bạn có thể dừng việc chờ đợi.

3.  **"Nhiệm vụ này có thông tin gì đặc biệt cần mang theo không?" (Request-scoped Values - Dữ liệu theo phạm vi request)**
    - **Tình huống:** Sau khi xác thực người dùng, bạn muốn tất cả các hàm trong chuỗi xử lý đều biết được `userID` của request hiện tại là gì.
    - **Cách làm:** Thay vì truyền `userID` như một tham số qua tất cả các hàm (`func DoSomething(param1, param2, userID string)`), bạn có thể "gắn" `userID` vào `context`. Các hàm sau đó có thể lấy `userID` ra từ `context` khi cần. (Lưu ý: Cách này nên được dùng một cách cẩn thận, chỉ cho các dữ liệu thực sự thuộc về request).

### `context` được truyền từ đâu?

Đây là phần quan trọng nhất đối với bạn:

**`context` được tạo ra bởi web framework (như Gin, Echo, hoặc `net/http` chuẩn) cho MỖI MỘT HTTP REQUEST đến.**

Nó bắt nguồn từ `Handler` và phải được **truyền tường minh** như một tham số đầu tiên xuống tất cả các lớp bên dưới (`Service`, `Repository`, ...).

#### Sơ đồ luồng đi của `context`:

```
[HTTP Request tới]
       |
       v
[Gin Handler]  <-- Framework tạo ra `ctx` ở đây (c.Request.Context())
       |
       |  func (h *AuthHandler) Login(c *gin.Context) {
       |      ctx := c.Request.Context() // Lấy context từ request
       |      h.authService.LoginWithGoogle(ctx, idToken) // Truyền ctx xuống Service
       |  }
       v
[Auth Service]
       |
       |  func (s *AuthService) LoginWithGoogle(ctx context.Context, idToken string) {
       |      s.VerifyGoogleIDToken(ctx, idToken) // Lại truyền ctx xuống tiếp
       |  }
       v
[Hàm VerifyToken]
       |
       |  func (s *AuthService) VerifyGoogleIDToken(ctx context.Context, idToken string) {
       |      // Thư viện của Google cũng yêu cầu ctx
       |      idtoken.Validate(ctx, idToken, s.googleClientID)
       |  }
       v
[Thư viện idtoken của Google] <-- Dùng ctx để có thể hủy request mạng nếu cần.
```

### Tóm lại

1.  **`context` là gì?** Là một đối tượng mang theo tín hiệu **hủy bỏ**, **deadline**, và các **dữ liệu theo phạm vi request**.
2.  **Mục đích?** Để kiểm soát vòng đời của một tác vụ, tránh lãng phí tài nguyên và làm cho ứng dụng của bạn mạnh mẽ, linh hoạt hơn.
3.  **Lấy từ đâu?** Web framework tạo ra nó cho bạn tại `Handler` (ví dụ `c.Request.Context()` trong Gin).
4.  **Dùng thế nào?** Luôn truyền nó như **tham số đầu tiên** xuống các lớp `Service`, `Repository`... trong suốt chuỗi xử lý của một request.

### PLantUML

```
@startuml
!theme vibrant

actor "Owner's App" as OwnerApp
actor "Viewer's App" as ViewerApp

package "Backend Monolith (Go)" {
    node "REST API (Ingestion)" as API
    node "Real-time Service (WebSocket)" as RealtimeSvc
    database "Database" as DB
}

queue "RabbitMQ" as MQ

OwnerApp -> API : POST /health-data (HTTP)
API -> DB : Save Health Data
API -> MQ : Publish "New Data for User A"

MQ --> RealtimeSvc : Consume "New Data for User A"
RealtimeSvc -> DB : Check who can view User A's data

ViewerApp <--> RealtimeSvc : WebSocket Connection (Persistent)

RealtimeSvc --> ViewerApp : Push "New Data for User A" (WebSocket)

' Luồng lấy dữ liệu lịch sử
ViewerApp --> API : GET /history/user_a (HTTP)
API -> DB : Query historical data
DB --> API
API --> ViewerApp

@enduml
```

dang cho phep dang nhap nhieu noi (khong thu hoi refresh khi dang nhap bang thiet bi khac)

## Phân tách `jwtauth` là một quyết định đúng đắn

Hãy xem xét vai trò của từng package:

---

### `jwtauth` (JWT Service)

- **Trách nhiệm duy nhất (Single Responsibility):**  
  Tạo và xác thực các chuỗi JWT.

- **Là một "công cụ" (Utility):**  
  Nó không nên biết bất cứ điều gì về logic nghiệp vụ.  
  Nó không cần biết "role là gì", "user là gì".  
  Nó chỉ cần biết rằng:

  > "Tôi nhận vào các thông tin (claims), và tôi mã hóa chúng thành một chuỗi token an toàn".

- **Sự phụ thuộc:**  
  Bằng cách truyền `email`, `id`, và `role` vào như các tham số chuỗi đơn giản, bạn đã giữ cho `jwtauth` hoàn toàn **độc lập**.  
  Nó không cần phải import `healthmate/internal/user`.  
  Điều này làm cho nó trở nên cực kỳ **dễ tái sử dụng**.  
  Bạn có thể **copy-paste** package `jwtauth` này sang một dự án hoàn toàn khác và nó vẫn hoạt động.

---

### `auth` (Authentication Service)

- **Trách nhiệm:**  
  Xử lý logic nghiệp vụ của việc xác thực.  
  Nó là người "nhạc trưởng".

- **Hiểu nghiệp vụ:**  
  Nó biết rằng một `user` có `email`, `id`, và `role`.

- **Sử dụng "công cụ":**  
  Trách nhiệm của nó là:
  > Lấy các thông tin nghiệp vụ này (`user.Email`, `user.ID`, `user.Role`)  
  > và đưa chúng cho "công cụ" `jwtauth` để tạo ra token.

---

### 1. Giải Thích Lại Luồng Ping/Pong và Thời Gian

Hãy tưởng tượng một cuộc điện thoại giữa `Server (writePump)` và `Client (readPump)` để kiểm tra xem đầu dây bên kia còn ai không.

- **`pongWait = 60 * time.Second` (Thời hạn chờ nghe):**

  - Đây là quy tắc của `readPump` (người nghe). Nó tự đặt ra một quy tắc: "Nếu trong 60 giây tới mà tôi không nghe thấy bất kỳ âm thanh nào (một tin nhắn PONG) từ client, tôi sẽ coi như client đã cúp máy."
  - Hành động: `c.conn.SetReadDeadline(time.Now().Add(pongWait))` chính là việc "đặt đồng hồ đếm ngược 60 giây".

- **`pingPeriod = 54 * time.Second` (Chu kỳ hỏi thăm):**

  - Đây là quy tắc của `writePump` (người nói). Nó tự đặt ra một quy tắc: "Cứ mỗi 54 giây, tôi phải chủ động nói 'Alo, còn đó không?' (gửi PING) để đảm bảo client vẫn còn nghe."
  - Hành động: `ticker := time.NewTicker(pingPeriod)` tạo ra một cái đồng hồ báo thức reo sau mỗi 54 giây.

- **`c.conn.SetPongHandler(...)` (Hành động khi nghe thấy PONG):**
  - Đây là quy tắc của `readPump`: "Khi tôi nghe thấy client trả lời 'Đây, vẫn còn nghe đây!' (nhận được PONG), tôi phải ngay lập tức reset lại đồng hồ đếm ngược 60 giây của mình."
  - Hành động: Hàm ẩn danh `func(...) { c.conn.SetReadDeadline(...) }` chính là hành động "reset đồng hồ".

**Luồng hoạt động hoàn chỉnh:**

1.  **Bắt đầu:** `readPump` đặt đồng hồ đếm ngược 60s. `writePump` đặt báo thức 54s.
2.  **Sau 54 giây:** Báo thức của `writePump` reo. Nó gửi một tin nhắn **PING** đến client.
3.  **Client nhận PING:** Thư viện WebSocket phía client (của trình duyệt, mobile app, hoặc Postman) **tự động và âm thầm nhận PING và ngay lập tức gửi lại một tin nhắn PONG**.
4.  **`readPump` nhận PONG:** `PongHandler` được kích hoạt. `readPump` reset lại đồng hồ đếm ngược của nó về 60s.
5.  **Vòng lặp tiếp tục:** Cứ mỗi 54 giây, `writePump` lại gửi PING, và `readPump` lại được reset đồng hồ.

**Kịch bản khi Client chết:**

1.  **Sau 54 giây:** `writePump` gửi PING.
2.  Client đã chết, không có ai trả lời PONG.
3.  **Sau 60 giây (kể từ lần nhận PONG cuối cùng):** Đồng hồ của `readPump` hết giờ. `c.conn.ReadJSON()` sẽ trả về một lỗi timeout.
4.  `readPump` thoát ra, `defer` được gọi, client được `unregister`.

---

### 2. Tại sao Postman không hiển thị tin nhắn PING/PONG?

Lý do là vì PING và PONG **không phải là tin nhắn dữ liệu (Data Messages)**. Chúng là các **Tin nhắn Điều khiển (Control Messages)**.

Giao thức WebSocket định nghĩa nhiều loại "khung" (frame) khác nhau có thể được gửi qua kết nối:

- **Text Frame:** Chứa dữ liệu dạng text (chuỗi JSON của bạn được gửi qua đây).
- **Binary Frame:** Chứa dữ liệu dạng nhị phân.
- **Ping Frame:** Tin nhắn điều khiển để kiểm tra kết nối.
- **Pong Frame:** Tin nhắn điều khiển để trả lời Ping.
- **Close Frame:** Tin nhắn điều khiển để đóng kết nối một cách lịch sự.

Hầu hết các giao diện người dùng của client WebSocket (như Postman, DevTools của trình duyệt) được thiết kế để chỉ hiển thị các **Data Messages** (Text và Binary) cho người dùng. Chúng tự động xử lý các **Control Messages** (Ping, Pong, Close) ở tầng dưới mà không làm phiền người dùng.

**Bạn đã hiểu đúng luồng hoạt động.** Tin nhắn PING thực sự đã được gửi đi từ server Go của bạn, và Postman đã nhận và tự động trả lời bằng PONG. Toàn bộ quá trình này diễn ra "trong suốt" (transparently) đối với bạn.

**Làm sao để chứng minh nó đang hoạt động?**

- Bạn có thể dùng một công cụ phân tích gói tin mạng cấp thấp như **Wireshark**. Nếu bạn bắt các gói tin trên cổng `8080`, bạn sẽ thấy rõ các WebSocket frame loại PING và PONG được gửi qua lại sau mỗi 54 giây.
- Một cách đơn giản hơn: Hãy thử đặt `pongWait` thành một giá trị rất nhỏ trong code, ví dụ `5 * time.Second`, và `pingPeriod` thành `10 * time.Second`. Sau đó kết nối. Bạn sẽ thấy sau khoảng 5 giây, server sẽ tự động ngắt kết nối của bạn vì nó không nhận được PONG kịp thời (do `writePump` chưa gửi PING). Điều này chứng tỏ cơ chế `Deadline` đang hoạt động.

// register thì user sẽ là pending, chờ xác minh otp email thành công thì mới thành active (user status có 3 trạng thái: pending, unverified, active, locked)

# Luồng Dữ Liệu Health Connect

## 1. Health Connect (Thiết bị người dùng)

- **Thu thập dữ liệu:** Nhịp tim, bước chân, calo.
- **Gửi dữ liệu:** Dữ liệu được gửi tới ứng dụng trên điện thoại.

## 2. App điện thoại / Backend

- **Xác thực & Kiểm tra quyền:** Xác thực người dùng và kiểm tra sự đồng ý (consent) về việc ai được phép xem dữ liệu.
- **Gắn Metadata:** Bổ sung các thông tin cần thiết như `user_id`, `timestamp`, `device_id`.
- **Đẩy vào Kafka:** Gửi sự kiện (event) vào topic Kafka `hc.vitals.raw`.

## 3. Kafka

- **Lưu trữ tạm thời:** Lưu trữ dữ liệu thô (raw data) một cách tạm thời.
- **Xử lý thời gian thực:** Cho phép các tiến trình stream processing đọc dữ liệu theo thời gian thực.

- **Các Topic chính:**
  - `hc.vitals.raw`: Chứa dữ liệu gốc từ thiết bị.
  - `hc.vitals.cleaned`: Dữ liệu sau khi đã được làm sạch.
  - `hc.vitals.aggregate`: Chứa các giá trị đã được tổng hợp (ví dụ: trung bình, min-max theo khoảng thời gian).
  - `hc.metadata.users`: Compacted topic, lưu trữ bản ghi mới nhất về consent và metadata của người dùng.
  - `hc.alerts`: Chứa các cảnh báo về chỉ số bất thường (ví dụ: nhịp tim).

## 4. Stream Processing

- **Tiền xử lý:** Lọc trùng lặp (deduplicate) và xác thực (validate) dữ liệu.
- **Làm giàu dữ liệu (Enrichment):** Kết hợp (join) với metadata từ topic `hc.metadata.users`.
- **Tính toán tổng hợp (Aggregation):** Tính toán các giá trị tổng hợp theo phút, giờ (ví dụ: trung bình).
- **Đầu ra:**
  - Xuất dữ liệu đã xử lý vào topic `hc.vitals.cleaned` và `hc.vitals.aggregate`.
  - Xuất cảnh báo vào topic `hc.alerts` nếu phát hiện bất thường (anomaly).

## 5. Database (TimescaleDB)

- **Lưu trữ lâu dài:** Lưu trữ dữ liệu sức khỏe để phân tích lịch sử.
- **Tối ưu cho chuỗi thời gian:** Sử dụng Hypertables để tối ưu việc truy vấn dữ liệu theo thời gian.

## 6. Realtime Push (WebSocket / SSE)

- **Consumer:** Đọc dữ liệu từ topic `hc.vitals.aggregate` hoặc `hc.vitals.cleaned`.
- **Đẩy dữ liệu real-time:** Push dữ liệu đến client của người được chia sẻ quyền (ví dụ: con cái xem dữ liệu của bố mẹ).
- **Kiểm tra quyền truy cập:** Luôn kiểm tra ACL (Access Control List) và consent trong thời gian thực trước khi đẩy dữ liệu.

# 2. Key Points về Consent và Quyền Truy Cập

- **Lưu trữ Consent:**
  - Thông tin về sự đồng ý (consent) được lưu trong database (đóng vai trò là "system of record").
  - Đồng thời được đẩy vào Kafka compacted topic (`hc.metadata.users`) để các tiến trình stream processing có thể join và kiểm tra quyền trong thời gian thực.
- **Cơ chế hoạt động:**
  - Người xem chỉ nhận được dữ liệu khi có sự đồng ý hợp lệ.
  - Người dùng có thể thu hồi (revoke) quyền truy cập bất cứ lúc nào. Khi đó, topic `hc.metadata.users` sẽ được cập nhật và pipeline xử lý sẽ ngay lập tức lọc dữ liệu dựa trên trạng thái consent mới nhất.

### Chạy Docker

```bash
docker run -d --name DACN-timescaledb -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=healthmate -v pgdata:/var/lib/postgresql/data -p 5432:5432 timescale/timescaledb:2.19.3-pg14

docker run -d --name DACN-redis -p 6379:6379 redis:8.2.0

docker run -d --name DACN-grafana -p 3000:3000 grafana/grafana:12.1.0

docker run -d --name DACN-prometheus -p 9090:9090 -v ./prometheus.yml:/etc/prometheus/prometheus.yml prom/prometheus:v3.6.0

docker run -d --name DACN-nginx -p 5001:5001 -v ./nginx/nginx.conf:/etc/nginx/conf.d/default.conf:ro nginx:1.28.0

```

xu ly khi database tat ma server van chay
