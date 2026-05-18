moi chac duoc lay otp, test duoc regitser email va verify account qua otp

so do class

```PLantUML
@startuml
!theme vibrant

title Sơ đồ Kiến trúc Tổng thể và Luồng Phụ thuộc


package "Main" {
    [main.go]
}

package "Application" {
    [App - DI Container] as App
}

package "Transport Layer (HTTP)" <<Rectangle>> {
    component "Gin Router" as Router
    component "AuthHandler" as Handler
    component "AuthMiddleware" as Middleware
}

package "Service Layer" <<Rectangle>>{
    interface IAuthService
    [AuthService] ..|> IAuthService

    interface ITokenService
    [TokenService] ..|> ITokenService
}

package "Data Access Layer" <<Rectangle>>{
    interface IUserRepository
    [UserRepository] ..|> IUserRepository
}

database "PostgreSQL" as DB
database "Redis" as Cache

' Relationships
main.go --> App : 1. Khởi tạo
App *-- Router
App *-- Handler
App *-- Middleware
App *-- AuthService
App *-- TokenService
App *-- UserRepository

Router ..> Handler : Sử dụng để xử lý route
Router ..> Middleware : Sử dụng để bảo vệ route

Handler ..> IAuthService : Phụ thuộc vào
Middleware ..> ITokenService : Phụ thuộc vào

AuthService ..> IUserRepository : Phụ thuộc vào
AuthService ..> ITokenService : Phụ thuộc vào

TokenService ..> Cache : (Tùy chọn) Dùng để revoke token
UserRepository ..> DB : Giao tiếp với DB
@enduml
```

kien truc he thong

```PLantUML
@startuml
!theme plain
skinparam shadowing false
skinparam actorStyle awesome
skinparam node {
    borderColor #006666
    borderThickness 2
}
skinparam database {
    borderColor #006666
    borderThickness 2
}
skinparam queue {
    borderColor #D35400
    borderThickness 2
}

title Sơ đồ kiến trúc hệ thống HealthMate

actor "User A" as UserA
agent "App của User A\n(Gửi dữ liệu)" as AppA
actor "User B\n(Người theo dõi)" as UserB
agent "App của User B\n(Nhận dữ liệu)" as AppB

cloud "Hệ thống Backend (Cloud Infrastructure)" {
    node "API Gateway (Go)" as Gateway
    queue "Apache Kafka" as Kafka

    package "Backend Services" {
        node "Storage Service\n(Consumer 1)" as StorageSvc
        node "Real-time Service\n(Consumer 2, WebSocket)" as RealtimeSvc
    }

    database "PostgreSQL Database" as DB {
        collections "TimescaleDB Hypertables\n(Dữ liệu Time-Series)" as Hypertables
        collections "Standard SQL Tables (Dữ liệu quan hệ)" as SQLTables
    }
}

' Data Ingestion Flow (Luồng thu thập dữ liệu)
UserA -right-> AppA
AppA -right-> Gateway : 1. Gửi dữ liệu (HTTPS)
Gateway -down-> Kafka : 2. Publish data

' Data Processing Flows (Luồng xử lý dữ liệu - Fan-out)
Kafka -right-> StorageSvc : 3a. Consume để lưu trữ
StorageSvc -down-> Hypertables : 4a. INSERT dữ liệu time-series

Kafka -down-> RealtimeSvc : 3b. Consume để đẩy real-time

' Real-time Delivery Flow (Luồng đẩy dữ liệu real-time)
UserB -left-> AppB
AppB <.left.> RealtimeSvc : 5. Mở kết nối WebSocket
RealtimeSvc .right.> SQLTables : 6. Kiểm tra quyền xem
RealtimeSvc -left-> AppB : 7. Đẩy dữ liệu mới

note right of RealtimeSvc
  Khi nhận dữ liệu từ Kafka:
  1. Tìm xem ai đang theo dõi user này.
  2. Đẩy dữ liệu qua kết nối WebSocket tương ứng.
end note

@enduml
```

### Core Features: Granular Sharing Controls

The system implements a sophisticated **"Group Rule as Base"** privacy model. This allows users to share health data with an entire group by default, while maintaining the ability to selectively restrict or grant access to specific individuals within that group. All services (`storage-service`, `realtime-service`) enforce these rules synchronously to ensure data privacy across both live streams and historical records.

````sql
id UUID PRIMARY KEY,
email VARCHAR(255) UNIQUE NOT NULL,
name VARCHAR(255) NOT NULL,
picture VARCHAR(255),
role VARCHAR(50) NOT NULL DEFAULT 'user',
status VARCHAR(50) NOT NULL DEFAULT 'unverified', -- "unverified", "verified"
provider VARCHAR(50) NOT NULL, -- "email", "google"
google_id VARCHAR(255) UNIQUE,
password_hash VARCHAR(255), -- Nên lưu hash, không lưu password thô
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

Để visualize cấu trúc dữ liệu và các quan hệ trong SQL từ một cơ sở dữ liệu, bạn có thể sử dụng một số công cụ trực tuyến miễn phí và phần mềm hỗ trợ như:

1. **dbdiagram.io**
   Đây là công cụ trực tuyến rất dễ sử dụng, cho phép bạn nhập schema SQL và nó tự động vẽ biểu đồ ERD (Entity Relationship Diagram) để bạn có thể nhìn thấy mối quan hệ giữa các bảng trong cơ sở dữ liệu.
   Website: [dbdiagram.io](https://dbdiagram.io)

2. **DrawSQL**
   Công cụ này tương tự dbdiagram.io, nhưng có tính năng hỗ trợ cộng tác và chia sẻ sơ đồ dễ dàng. Bạn có thể import SQL schema và DrawSQL sẽ giúp bạn tạo ERD.
   Website: [DrawSQL](https://drawsql.app)

3. **SQLDBM**
   SQLDBM là một công cụ online mạnh mẽ cho phép bạn tạo ERD từ SQL schema, và có khả năng tự động hóa việc thiết kế cơ sở dữ liệu. Nó hỗ trợ nhiều loại cơ sở dữ liệu khác nhau như MySQL, PostgreSQL, MS SQL Server.
   Website: [SQLDBM](https://sqldbm.com)

4. **MySQL Workbench (Desktop)**
   Nếu bạn muốn một công cụ desktop, MySQL Workbench là lựa chọn tuyệt vời. Nó có tính năng "Reverse Engineer" giúp bạn tạo mô hình ERD từ cơ sở dữ liệu SQL hiện có.
   Website: [MySQL Workbench](https://dev.mysql.com/downloads/workbench/)

5. **Vertabelo**
   Vertabelo là một công cụ trực tuyến khác cho phép thiết kế và visual hóa cơ sở dữ liệu. Bạn có thể nhập SQL script và nó sẽ giúp bạn tạo ra một mô hình ERD đẹp mắt.
   Website: [Vertabelo](https://www.vertabelo.com)

Các công cụ này đều có khả năng giúp bạn "visualize" cấu trúc cơ sở dữ liệu của SQL, cũng như các mối quan hệ giữa các bảng, từ đó dễ dàng hiểu và quản lý hơn. Nếu bạn có một cơ sở dữ liệu cụ thể hoặc mã SQL cần hỗ trợ, tôi có thể giúp bạn thêm chi tiết!

loi moi minh da gui

```sql
SELECT
    invited_user.name AS invited_user_name,
    invited_user.email AS invited_user_email,
    inviter.name AS inviter_name,
    gm.status,
    gm.created_at AS invitation_sent_at
FROM
    group_members AS gm
JOIN
    users AS invited_user ON gm.user_id = invited_user.id
JOIN
    users AS inviter ON gm.invited_by = inviter.id
WHERE
    gm.group_id = :your_group_id AND gm.status = 'pending';
````

loi moi minh nhan duoc

```sql
SELECT
    g.name AS group_name,
    inviter.name AS inviter_name,
    gm.created_at AS invitation_received_at
FROM
    group_members AS gm
JOIN
    groups AS g ON gm.group_id = g.id
JOIN
    users AS inviter ON gm.invited_by = inviter.id
WHERE
    gm.user_id = :current_user_id AND gm.status = 'pending';
```

```
HealthMate
├─ .agent
│  ├─ rules
│  │  └─ healthmate.md
│  ├─ skills
│  │  ├─ database
│  │  │  └─ add_database_migration.md
│  │  ├─ debugging
│  │  │  └─ debug_request_flow.md
│  │  ├─ errors
│  │  │  └─ add_domain_error.md
│  │  ├─ feature
│  │  │  ├─ add_new_service_module.md
│  │  │  ├─ implement_new_feature.md
│  │  │  └─ register_new_route.md
│  │  └─ testing
│  │     └─ write_service_unit_tests.md
│  └─ skills.yml
├─ .claude
│  ├─ settings.json
│  └─ settings.local.json
├─ api-gateway
│  ├─ app
│  │  ├─ app.go
│  │  └─ router.go
│  ├─ cmd
│  │  └─ server
│  │     └─ main.go
│  ├─ config
│  │  └─ config.go
│  ├─ Dockerfile
│  ├─ go.mod
│  ├─ go.sum
│  └─ internal
│     ├─ handlers
│     │  ├─ health.go
│     │  ├─ kafka.go
│     │  └─ proxy.go
│     ├─ kafka
│     │  └─ producer.go
│     └─ middleware
│        ├─ auth.go
│        ├─ cors.go
│        └─ ratelimit.go
├─ API_DOC.md
├─ auth-service
│  ├─ app
│  │  └─ app.go
│  ├─ cmd
│  │  └─ server
│  │     └─ main.go
│  ├─ config
│  │  └─ config.go
│  ├─ Dockerfile
│  ├─ docs
│  │  ├─ docs.go
│  │  ├─ swagger.json
│  │  └─ swagger.yaml
│  ├─ go.mod
│  ├─ go.sum
│  ├─ internal
│  │  ├─ auth
│  │  │  ├─ auth_service.go
│  │  │  ├─ auth_service_test.go
│  │  │  ├─ error.go
│  │  │  ├─ google_verifier.go
│  │  │  ├─ handler.go
│  │  │  ├─ jwt_token_service.go
│  │  │  ├─ jwt_token_service_test.go
│  │  │  ├─ otp_service.go
│  │  │  ├─ redis_otp_service.go
│  │  │  ├─ redis_otp_service_test.go
│  │  │  └─ token_service.go
│  │  ├─ cache
│  │  │  └─ cache.go
│  │  ├─ common
│  │  │  ├─ context_keys.go
│  │  │  └─ errors.go
│  │  ├─ domain
│  │  │  ├─ group.go
│  │  │  ├─ group_memeber.go
│  │  │  ├─ guser.go
│  │  │  ├─ metric_type.go
│  │  │  ├─ otp.go
│  │  │  ├─ permission.go
│  │  │  └─ user.go
│  │  ├─ group
│  │  │  ├─ errors.go
│  │  │  ├─ group_repository.go
│  │  │  ├─ group_service.go
│  │  │  ├─ group_service_test.go
│  │  │  ├─ handler.go
│  │  │  └─ postgres_group_repository.go
│  │  ├─ mail
│  │  │  ├─ email_service.go
│  │  │  └─ gmail_service.go
│  │  ├─ member
│  │  │  ├─ errors.go
│  │  │  ├─ member_repository.go
│  │  │  ├─ member_service.go
│  │  │  ├─ member_service_test.go
│  │  │  └─ postgres_member_repository.go
│  │  ├─ permission
│  │  │  ├─ errors.go
│  │  │  ├─ permission_repository.go
│  │  │  ├─ permission_service.go
│  │  │  ├─ permission_service_test.go
│  │  │  └─ postgres_permission_repository.go
│  │  ├─ platform
│  │  │  ├─ postgres
│  │  │  │  └─ client.go
│  │  │  └─ redis
│  │  │     ├─ client.go
│  │  │     └─ redis_cache.go
│  │  ├─ user
│  │  │  ├─ errors.go
│  │  │  ├─ handler.go
│  │  │  ├─ postgres_user_repository.go
│  │  │  ├─ user_repository.go
│  │  │  ├─ user_service.go
│  │  │  └─ user_service_test.go
│  │  ├─ validation
│  │  │  ├─ adapter.go
│  │  │  └─ validator.go
│  │  └─ web
│  │     ├─ helpers
│  │     │  ├─ context_helpers.go
│  │     │  ├─ request_helpers.go
│  │     │  ├─ response_helpers.go
│  │     │  └─ types.go
│  │     └─ middleware
│  │        ├─ error.go
│  │        ├─ group_middleware.go
│  │        ├─ prometheus.go
│  │        └─ rbac.go
│  ├─ mocks
│  │  ├─ Cache.go
│  │  ├─ EmailService.go
│  │  ├─ GoogleTokenVerifier.go
│  │  ├─ GroupChecker.go
│  │  ├─ GroupRepository.go
│  │  ├─ MemberChecker.go
│  │  ├─ MemberRepository.go
│  │  ├─ MemberRepositoryInterface.go
│  │  ├─ OTPService.go
│  │  ├─ PermissionRepository.go
│  │  ├─ scannable.go
│  │  ├─ Service.go
│  │  ├─ TokenService.go
│  │  ├─ UserChecker.go
│  │  └─ UserRepository.go
│  └─ README.md
├─ auth_manual_test.md
├─ cors-debug-report.md
├─ dac-ta-lich-thuoc.md
├─ deploy
│  └─ docker-compose.prod.yml
├─ deploy.md
├─ docker-compose.yml
├─ dump.sql
├─ group-workflow-test.html
├─ GUIDE.md
├─ kafka
│  ├─ docker-compose.yml
│  └─ kui
│     └─ config.yml
├─ learn.md
├─ models
│  ├─ hgbr_health_model.joblib
│  ├─ readiness_model.onnx
│  ├─ stress_catboost.joblib
│  └─ stress_hgbc.joblib
├─ nginx
│  └─ nginx.conf
├─ ocr-service
│  ├─ app
│  │  ├─ allergy_hints.py
│  │  ├─ constants.py
│  │  ├─ drug_dictionary.py
│  │  ├─ fallback.py
│  │  ├─ main.py
│  │  ├─ ocr_engine.py
│  │  ├─ parser.py
│  │  ├─ postprocess.py
│  │  ├─ preprocess.py
│  │  ├─ privacy_redact.py
│  │  ├─ settings.py
│  │  └─ template_hints.py
│  ├─ data
│  │  └─ drugs_moh.csv
│  ├─ Dockerfile
│  ├─ README.md
│  ├─ requirements.txt
│  └─ scripts
│     ├─ benchmark.py
│     ├─ build_drug_dict_from_medicine_data.py
│     ├─ dataset
│     │  ├─ images
│     │  │  ├─ 041fec48-0760-48d5-a8fb-73b72843c8c7.jpg
│     │  │  ├─ 0760c8f6-adbd-4254-9418-a62ba6d6916d.jpg
│     │  │  ├─ 0958ffa1-a1ca-4f4b-bd6f-c4128bf2dfbf.jpg
│     │  │  ├─ 1a6c2ce3-0bf8-49b9-91fa-a60141e8d482.jpg
│     │  │  ├─ 25346b08-6c91-4074-90ab-8dec3390f259.jpg
│     │  │  ├─ 2c68f313-fbbd-4141-a079-77722646fc15.jpg
│     │  │  ├─ 2d4cf12c-aed7-495f-8d0e-20c423a91b75.jpg
│     │  │  ├─ 2fcc5be2-d1f8-47eb-929b-aa36e46a8261.jpg
│     │  │  ├─ 34f388e2-6d7b-4f0d-8de8-23ac3de37c51.jpg
│     │  │  ├─ 36b5fb0d-6b33-44b6-b187-387f9f86c920.jpg
│     │  │  ├─ 544a2e2e-9f19-4d68-9f6f-cb018dd83c91.jpg
│     │  │  ├─ 603e8b0b-e59f-4b87-9bb9-e19d02c39984.jpg
│     │  │  ├─ 642339050_1455056389447802_3264942585587069227_n.jpg
│     │  │  ├─ 659813677_2416353085548154_7601677991925691977_n.jpg
│     │  │  ├─ 660337762_1950367022512887_6912860012122313822_n.jpg
│     │  │  ├─ 661072901_962564969521177_3641818443280458009_n.jpg
│     │  │  ├─ 662169494_948940104493450_1702267021556638530_n.jpg
│     │  │  ├─ 662250004_2172952129911482_1433779290536495622_n.jpg
│     │  │  ├─ 662271561_4856094791283499_1664676231828565317_n.jpg
│     │  │  ├─ 662417215_2333657857143140_1860711517623749621_n.jpg
│     │  │  ├─ 662618620_954224710670549_9132235593096998975_n.jpg
│     │  │  ├─ 664022051_869372206115407_2643172541354722065_n.jpg
│     │  │  ├─ 665418263_913518704886739_6668465618317626612_n.jpg
│     │  │  ├─ 665533573_2131369121039200_493664280806630949_n.jpg
│     │  │  ├─ 666239886_1593150598418746_8153015708731970082_n.jpg
│     │  │  ├─ 666322264_1497815005076536_8018140414727040859_n.jpg
│     │  │  ├─ 667589965_972644985418269_9138318291086751492_n.jpg
│     │  │  ├─ 6b9df5dd-6510-4a1c-96f5-a69526c0dcfa.jpg
│     │  │  ├─ 7bd21d98-39a1-424f-9b1a-2b4db6bc8925.jpg
│     │  │  ├─ 7cdd54a8-151a-48f0-9d62-1531384ae132.jpg
│     │  │  ├─ 84049047-f768-469f-a580-b563d8d19cb2.jpg
│     │  │  ├─ 862f2381-8882-434b-bf1d-b66d9392a786.jpg
│     │  │  ├─ 86d1b5f1-bfef-493a-bc3d-7f825369b699.jpg
│     │  │  ├─ 878a97d2-0c8c-436e-955f-bb6917bf6fc4.jpg
│     │  │  ├─ a4130ac0-7f22-42de-8a77-c8c32aeb1c96.jpg
│     │  │  ├─ ba54a32d-4ca9-4103-8787-f9846be11c8d.jpg
│     │  │  ├─ bb9fc860-8614-415d-ad5a-ff4244949970.jpg
│     │  │  ├─ c62fec48-627d-434b-8adb-1fbf0b55cb43.jpg
│     │  │  ├─ c8eac591-e5c2-4272-aeda-8a6b2d7d55a7.jpg
│     │  │  ├─ cc0a1d5a-0f35-4e8e-b509-058fa380f64a.jpg
│     │  │  ├─ d45b04f2-fa9d-44c7-b81d-e77490f1d0bd.jpg
│     │  │  ├─ d49a8839-d228-4e06-a18f-2965e767ed90.jpg
│     │  │  ├─ d63c5598-bb2b-4306-894c-251c4cedf71c.jpg
│     │  │  ├─ d7abcd2e-cda5-4747-ba26-920c377189fb.jpg
│     │  │  ├─ dc11d126-6f5e-4e43-9979-1677dbb66d49.jpg
│     │  │  ├─ e3a15b9e-975a-4199-9e73-566a682f4171.jpg
│     │  │  ├─ fcd224ee-ab88-4668-8947-761c94ec14fe.jpg
│     │  │  ├─ z7748543907384_d894078b0b5589374f809954ffb1a97c.jpg
│     │  │  ├─ z7748543914107_e4757cb2180230b203c0730a121a8747.jpg
│     │  │  ├─ z7748543922077_90c3ff87f52860e3eaa85ccba305dda3.jpg
│     │  │  ├─ z7748543927444_f6233b6acffdd8493ef8f5763aff782b.jpg
│     │  │  ├─ z7748543940927_0ab476229364a014dc328998e1fd389a.jpg
│     │  │  ├─ z7748543946856_0da9e51d4cfc773b7ba8447ca74e95c4.jpg
│     │  │  ├─ z7748543950320_1de0413799126d2c412c013b48a5da22.jpg
│     │  │  ├─ z7748543963758_9cd98091e2edf30c541ce929f4d7ae70.jpg
│     │  │  ├─ z7748543973504_63acfd7be79ddb915ba70a2db3b08895.jpg
│     │  │  ├─ z7748543976949_66d98c872909c0172cfea95a1f943b3c.jpg
│     │  │  ├─ z7748543986280_b57455021aeb067deb53dd48f784a58c.jpg
│     │  │  ├─ z7748543987211_e2e3dbe241d5cc47a5a3951438c25e25.jpg
│     │  │  ├─ z7748543999280_94ba4464165e5cabb673421e65c2aa99.jpg
│     │  │  ├─ z7748544003418_a2702d5a688f928e5267e03562e52ea2.jpg
│     │  │  ├─ z7748544014144_7da2e5036755967f7848dcbe452c7809.jpg
│     │  │  ├─ z7748544017410_4cb276fbbe46ee579f3777fb91c0c85b.jpg
│     │  │  ├─ z7748544025474_5f45dd65302c60b57552fb89b9d368a8.jpg
│     │  │  ├─ z7748544036650_b43da6b4a791022727c92dc1b8339b2c.jpg
│     │  │  └─ z7748544046550_3f71bacf4c76a71e73226b5ea0a400aa.jpg
│     │  └─ labels
│     │     ├─ 041fec48-0760-48d5-a8fb-73b72843c8c7.json
│     │     ├─ 0760c8f6-adbd-4254-9418-a62ba6d6916d.json
│     │     ├─ 0958ffa1-a1ca-4f4b-bd6f-c4128bf2dfbf.json
│     │     ├─ 1a6c2ce3-0bf8-49b9-91fa-a60141e8d482.json
│     │     ├─ 25346b08-6c91-4074-90ab-8dec3390f259.json
│     │     ├─ 2c68f313-fbbd-4141-a079-77722646fc15.json
│     │     ├─ 2d4cf12c-aed7-495f-8d0e-20c423a91b75.json
│     │     ├─ 2fcc5be2-d1f8-47eb-929b-aa36e46a8261.json
│     │     ├─ 34f388e2-6d7b-4f0d-8de8-23ac3de37c51.json
│     │     ├─ 36b5fb0d-6b33-44b6-b187-387f9f86c920.json
│     │     ├─ 544a2e2e-9f19-4d68-9f6f-cb018dd83c91.json
│     │     ├─ 603e8b0b-e59f-4b87-9bb9-e19d02c39984.json
│     │     ├─ 642339050_1455056389447802_3264942585587069227_n.json
│     │     ├─ 659813677_2416353085548154_7601677991925691977_n.json
│     │     ├─ 660337762_1950367022512887_6912860012122313822_n.json
│     │     ├─ 661072901_962564969521177_3641818443280458009_n.json
│     │     ├─ 662169494_948940104493450_1702267021556638530_n.json
│     │     ├─ 662250004_2172952129911482_1433779290536495622_n.json
│     │     ├─ 662271561_4856094791283499_1664676231828565317_n.json
│     │     ├─ 662417215_2333657857143140_1860711517623749621_n.json
│     │     ├─ 662618620_954224710670549_9132235593096998975_n.json
│     │     ├─ 664022051_869372206115407_2643172541354722065_n.json
│     │     ├─ 665418263_913518704886739_6668465618317626612_n.json
│     │     ├─ 665533573_2131369121039200_493664280806630949_n.json
│     │     ├─ 666239886_1593150598418746_8153015708731970082_n.json
│     │     ├─ 666322264_1497815005076536_8018140414727040859_n.json
│     │     ├─ 667589965_972644985418269_9138318291086751492_n.json
│     │     ├─ 6b9df5dd-6510-4a1c-96f5-a69526c0dcfa.json
│     │     ├─ 7bd21d98-39a1-424f-9b1a-2b4db6bc8925.json
│     │     ├─ 7cdd54a8-151a-48f0-9d62-1531384ae132.json
│     │     ├─ 84049047-f768-469f-a580-b563d8d19cb2.json
│     │     ├─ 862f2381-8882-434b-bf1d-b66d9392a786.json
│     │     ├─ 86d1b5f1-bfef-493a-bc3d-7f825369b699.json
│     │     ├─ 878a97d2-0c8c-436e-955f-bb6917bf6fc4.json
│     │     ├─ a4130ac0-7f22-42de-8a77-c8c32aeb1c96.json
│     │     ├─ ba54a32d-4ca9-4103-8787-f9846be11c8d.json
│     │     ├─ bb9fc860-8614-415d-ad5a-ff4244949970.json
│     │     ├─ c62fec48-627d-434b-8adb-1fbf0b55cb43.json
│     │     ├─ c8eac591-e5c2-4272-aeda-8a6b2d7d55a7.json
│     │     ├─ cc0a1d5a-0f35-4e8e-b509-058fa380f64a.json
│     │     ├─ d45b04f2-fa9d-44c7-b81d-e77490f1d0bd.json
│     │     ├─ d49a8839-d228-4e06-a18f-2965e767ed90.json
│     │     ├─ d63c5598-bb2b-4306-894c-251c4cedf71c.json
│     │     ├─ d7abcd2e-cda5-4747-ba26-920c377189fb.json
│     │     ├─ dc11d126-6f5e-4e43-9979-1677dbb66d49.json
│     │     ├─ e3a15b9e-975a-4199-9e73-566a682f4171.json
│     │     ├─ fcd224ee-ab88-4668-8947-761c94ec14fe.json
│     │     ├─ z7748543907384_d894078b0b5589374f809954ffb1a97c.json
│     │     ├─ z7748543914107_e4757cb2180230b203c0730a121a8747.json
│     │     ├─ z7748543922077_90c3ff87f52860e3eaa85ccba305dda3.json
│     │     ├─ z7748543927444_f6233b6acffdd8493ef8f5763aff782b.json
│     │     ├─ z7748543940927_0ab476229364a014dc328998e1fd389a.json
│     │     ├─ z7748543946856_0da9e51d4cfc773b7ba8447ca74e95c4.json
│     │     ├─ z7748543950320_1de0413799126d2c412c013b48a5da22.json
│     │     ├─ z7748543963758_9cd98091e2edf30c541ce929f4d7ae70.json
│     │     ├─ z7748543973504_63acfd7be79ddb915ba70a2db3b08895.json
│     │     ├─ z7748543976949_66d98c872909c0172cfea95a1f943b3c.json
│     │     ├─ z7748543986280_b57455021aeb067deb53dd48f784a58c.json
│     │     ├─ z7748543987211_e2e3dbe241d5cc47a5a3951438c25e25.json
│     │     ├─ z7748543999280_94ba4464165e5cabb673421e65c2aa99.json
│     │     ├─ z7748544003418_a2702d5a688f928e5267e03562e52ea2.json
│     │     ├─ z7748544014144_7da2e5036755967f7848dcbe452c7809.json
│     │     ├─ z7748544017410_4cb276fbbe46ee579f3777fb91c0c85b.json
│     │     ├─ z7748544025474_5f45dd65302c60b57552fb89b9d368a8.json
│     │     ├─ z7748544036650_b43da6b4a791022727c92dc1b8339b2c.json
│     │     ├─ z7748544046550_3f71bacf4c76a71e73226b5ea0a400aa.json
│     │     └─ _generation_report.json
│     ├─ diagnose.py
│     ├─ dump_moh_full.py
│     ├─ ocr_service_eval.executed.ipynb
│     ├─ ocr_service_eval.ipynb
│     └─ reports
│        ├─ benchmark_official_20260508_004721_per_image.csv
│        ├─ benchmark_official_20260508_004721_summary.json
│        ├─ ocr_eval_per_image.csv
│        └─ ocr_eval_summary.json
├─ ocr-test.html
├─ rate_limiting_summary.md
├─ README.md
├─ realtime-service
│  ├─ app
│  │  └─ app.go
│  ├─ cmd
│  │  └─ server
│  │     └─ main.go
│  ├─ config
│  │  └─ config.go
│  ├─ Dockerfile
│  ├─ docs
│  │  ├─ docs.go
│  │  ├─ swagger.json
│  │  └─ swagger.yaml
│  ├─ go.mod
│  ├─ go.sum
│  ├─ internal
│  │  ├─ auth
│  │  │  ├─ jwt_validator.go
│  │  │  └─ jwt_validator_test.go
│  │  ├─ common
│  │  │  └─ errors.go
│  │  ├─ kafka
│  │  │  ├─ consumer.go
│  │  │  └─ producer.go
│  │  ├─ metric
│  │  │  └─ metric.go
│  │  ├─ permission
│  │  │  ├─ errors.go
│  │  │  ├─ permission_repository.go
│  │  │  └─ postgres_permission_repository.go
│  │  ├─ platform
│  │  │  └─ postgres
│  │  │     └─ client.go
│  │  └─ realtime
│  │     ├─ client.go
│  │     ├─ client_test.go
│  │     ├─ handler.go
│  │     ├─ handler_test.go
│  │     ├─ hub.go
│  │     └─ hub_test.go
│  └─ README.md
├─ realtime_integration_test.md
├─ report.md
├─ scripts
│  ├─ demo_push_realtime_metrics.py
│  ├─ demo_realtime_timescale_seed.sql
│  ├─ e2e_realtime_ws_test.py
│  ├─ e2e_requirements.txt
│  ├─ e2e_sharing_permissions_seed.sql
│  ├─ e2e_sharing_permissions_test.py
│  ├─ README_E2E_SHARING.md
│  └─ tools
│     └─ gen_bcrypt.go
└─ storage-service
   ├─ app
   │  └─ app.go
   ├─ cmd
   │  └─ server
   │     └─ main.go
   ├─ config
   │  ├─ config.go
   │  └─ firebase-service-account.json
   ├─ Dockerfile
   ├─ docs
   │  ├─ docs.go
   │  ├─ swagger.json
   │  └─ swagger.yaml
   ├─ go.mod
   ├─ go.sum
   ├─ internal
   │  ├─ common
   │  │  └─ errors.go
   │  ├─ kafka
   │  │  ├─ consumer.go
   │  │  └─ message.go
   │  ├─ medication
   │  │  ├─ errors.go
   │  │  ├─ handler.go
   │  │  ├─ medication.go
   │  │  ├─ medication_service_test.go
   │  │  ├─ repository.go
   │  │  ├─ scheduler.go
   │  │  └─ service.go
   │  ├─ metric
   │  │  ├─ errors.go
   │  │  ├─ metric.go
   │  │  ├─ metric_handler.go
   │  │  ├─ metric_repository.go
   │  │  ├─ metric_servcie.go
   │  │  ├─ metric_service_test.go
   │  │  ├─ threshold.go
   │  │  └─ timescaledb_metric_repository.go
   │  ├─ middleware
   │  │  ├─ jwt_auth.go
   │  │  └─ jwt_auth_test.go
   │  ├─ notification
   │  │  ├─ notification.go
   │  │  ├─ repository.go
   │  │  └─ service.go
   │  ├─ platform
   │  │  └─ postgres
   │  │     └─ client.go
   │  ├─ readiness
   │  │  ├─ handler.go
   │  │  └─ predictor.go
   │  └─ web
   │     └─ helpers
   │        └─ response_helpers.go
   ├─ migration
   │  ├─ 000001_create_users_table.down.sql
   │  ├─ 000001_create_users_table.up.sql
   │  ├─ 000002_update_users_table.down.sql
   │  ├─ 000002_update_users_table.up.sql
   │  ├─ 000003_create_group_table.down.sql
   │  ├─ 000003_create_group_table.up.sql
   │  ├─ 000004_create_permission_table.down.sql
   │  ├─ 000004_create_permission_table.up.sql
   │  ├─ 000005_create_metric_table.down.sql
   │  ├─ 000005_create_metric_table.up.sql
   │  ├─ 000006_create_metric_types_table.down.sql
   │  ├─ 000006_create_metric_types_table.up.sql
   │  ├─ 000007_alter_metric_types.down.sql
   │  ├─ 000007_alter_metric_types.up.sql
   │  ├─ 000008_add_user_profile_fields.down.sql
   │  ├─ 000008_add_user_profile_fields.up.sql
   │  ├─ 000009_add_spo2_blood_pressure.down.sql
   │  ├─ 000009_add_spo2_blood_pressure.up.sql
   │  ├─ 000010_create_medication_reminders_table.down.sql
   │  ├─ 000010_create_medication_reminders_table.up.sql
   │  ├─ 000011_create_user_device_tokens_table.down.sql
   │  ├─ 000011_create_user_device_tokens_table.up.sql
   │  ├─ 000012_create_user_thresholds_table.down.sql
   │  ├─ 000012_create_user_thresholds_table.up.sql
   │  ├─ 000013_add_shared_with_to_permissions.down.sql
   │  ├─ 000013_add_shared_with_to_permissions.up.sql
   │  ├─ 000014_medication_sharing.down.sql
   │  ├─ 000014_medication_sharing.up.sql
   │  ├─ 000015_add_timezone_to_users.down.sql
   │  ├─ 000015_add_timezone_to_users.up.sql
   │  ├─ 000016_remove_timezone_from_medications.down.sql
   │  └─ 000016_remove_timezone_from_medications.up.sql
   ├─ mocks
   │  ├─ MedicationRepository.go
   │  ├─ MetricRepository.go
   │  ├─ NotificationRepository.go
   │  └─ NotificationService.go
   ├─ models
   │  ├─ readiness_model.onnx
   │  └─ readiness_model.onnx.bak
   └─ README.md

```