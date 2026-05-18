TimescaleDB được chọn làm nền tảng Database cốt lõi (sử dụng trên PostgreSQL) vì nó được tối ưu hóa đặc biệt cho dữ liệu chuỗi thời gian (time-series data) – loại dữ liệu chính của dự án. Sự lựa chọn này dựa trên ba tính năng kiến trúc chính giải quyết các thách thức về lưu trữ, tốc độ ghi (ingest), và hiệu suất truy vấn phân tích lịch sử.

### 2.1. Hypertables (Siêu Bảng) – Quản lý Dữ liệu Hiệu quả

TimescaleDB sử dụng **Hypertables** để tự động phân vùng (partition) dữ liệu chuỗi thời gian theo thời gian.

- **Phân vùng tự động:** Dữ liệu được chia thành các phân vùng con gọi là **chunks** (khối). Mặc định, mỗi chunk giữ dữ liệu trong 7 ngày, nhưng có thể điều chỉnh theo nhu cầu (ví dụ: 1 ngày).
- **Tối ưu hóa Truy vấn Lịch sử:** Khi người dùng truy vấn lịch sử (Luồng Đọc Dữ liệu), TimescaleDB thực hiện **Chunk Skipping** (bỏ qua khối). Cơ chế này chỉ xác định và chạy truy vấn trên các chunk chứa dữ liệu liên quan, thay vì quét toàn bộ bảng khổng lồ, giúp **giảm đáng kể thời gian và tài nguyên** cần thiết để truy xuất kết quả.
- Hypertables cũng có thể phân vùng theo các chiều khác (dimensions), ví dụ như theo `device_id` (ID thiết bị) bằng cách sử dụng Hash Partitioning.

### 2.2. Hypercore – Engine Lưu trữ Lai (Row-Columnar)

Hypercore là một engine lưu trữ lai (hybrid row-columnar storage engine) được thiết kế đặc biệt cho phân tích real-time dữ liệu chuỗi thời gian. Cơ chế này giải quyết yêu cầu về khả năng chịu tải cao và hiệu suất truy vấn nhanh:

| Thành phần Hypercore               | Chức năng và Lợi ích                                                                                                                                                        | Ứng dụng trong Dự án                                                                                                          |
| :--------------------------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :---------------------------------------------------------------------------------------------------------------------------- |
| **Rowstore (Lưu trữ theo Hàng)**   | Tối ưu hóa cho tốc độ chèn (insert) và cập nhật cao. Đảm bảo **High ingest throughput** và **Low-latency ingestion**. Hỗ trợ đầy đủ Mutability (UPSERTs, UPDATES, DELETEs). | Hỗ trợ **Storage Service** xử lý luồng dữ liệu đến liên tục và nhanh chóng từ Kafka (Cold Path).                              |
| **Columnstore (Lưu trữ theo Cột)** | Dữ liệu "nguội" được tự động chuyển đổi sang định dạng cột. Định dạng này tối ưu hóa hiệu suất cho **các tác vụ phân tích và tổng hợp**.                                    | Tăng tốc độ truy vấn lịch sử và phân tích (Data Retrieval Flow). Giúp **tiết kiệm đáng kể không gian lưu trữ** (nén tới 98%). |

**Tính nhất quán:** Dù dữ liệu được lưu trữ ở Rowstore hay Columnstore, Hypercore vẫn cung cấp hỗ trợ **ACID** đầy đủ, đảm bảo tính nhất quán giao dịch.

### 2.3. Continuous Aggregates (CAs) – Tăng tốc Phân tích Real-time

Continuous Aggregates là một tính năng độc quyền của TimescaleDB, được sử dụng để tạo các bản tóm tắt dữ liệu một cách cực kỳ nhanh chóng.

- **Tự động và Gia tăng (Incremental):** CAs là một loại hypertable được **làm mới tự động trong nền** khi dữ liệu mới được thêm hoặc dữ liệu cũ được sửa đổi. Khác với Materialized Views thông thường của Postgres (phải tạo lại toàn bộ), CAs được cập nhật **liên tục và gia tăng**, do đó yêu cầu bảo trì và tài nguyên ít hơn nhiều.
- **Tăng tốc Truy vấn Phân tích:** Khi người dùng yêu cầu biểu đồ lịch sử (ví dụ: trung bình nhịp tim hàng ngày/hàng giờ), CAs đã tính toán trước các giá trị tổng hợp này, tránh việc quét và tính toán lại dữ liệu thô khổng lồ.
- **Kết quả Real-time:** CAs có khả năng kết hợp dữ liệu đã được tổng hợp trước đó với **dữ liệu thô mới nhất** để cung cấp kết quả **chính xác và cập nhật tức thời** cho mỗi truy vấn (Real-time Aggregation).
- **Hỗ trợ JOIN phức tạp:** CAs hỗ trợ các câu lệnh `JOIN` (như `INNER JOIN`, `LEFT JOIN`, và `LATERAL JOIN`) giữa hypertable và các bảng Postgres tiêu chuẩn. Điều này quan trọng để liên kết dữ liệu sức khỏe (time-series) với dữ liệu người dùng/thiết bị (Postgres tables) trong Luồng Đọc Dữ liệu.

---

**Kết luận:** Sự kết hợp giữa **Hypertables** (phân vùng hiệu quả và Chunk Skipping), **Hypercore** (tốc độ ghi cao và nén dữ liệu phân tích), và **Continuous Aggregates** (phân tích lịch sử nhanh, cập nhật real-time) khiến TimescaleDB trở thành lựa chọn lý tưởng, đáp ứng đồng thời cả nhu cầu lưu trữ bền vững (Cold Path) và nhu cầu phân tích dữ liệu lịch sử tốc độ cao (Data Retrieval) của dự án Theo dõi Sức khỏe Real-time.

**BÁO CÁO NGHIÊN CỨU KIẾN TRÚC HỆ THỐNG**

**Giới Thiệu và Phân Tích Cơ Sở Dữ Liệu Chuyên Dụng TimescaleDB (Tiger Data)**
**Trong Dự Án Theo Dõi Sức Khỏe Real-time (Real-time Health Monitoring)**

### 1. GIỚI THIỆU VỀ TIMESCALEDB VÀ DỮ LIỆU CHUỖI THỜI GIAN

TimescaleDB là một nền tảng cơ sở dữ liệu mạnh mẽ được xây dựng để phục vụ phân tích thời gian thực trên dữ liệu chuỗi thời gian, tích hợp liền mạch với hệ sinh thái PostgreSQL thông qua việc mở rộng (extension). Không giống như các hệ thống phân tích truyền thống dựa vào xử lý theo lô (batch processing) và báo cáo chậm trễ, TimescaleDB được thiết kế để xử lý và truy vấn dữ liệu ngay khi nó được tạo ra và tích lũy, cung cấp thông tin chi tiết tức thời và liên tục.

Hệ thống TimescaleDB được tối ưu hóa để đáp ứng các yêu cầu khắt khe của phân tích thời gian thực, bao gồm: truy vấn độ trễ thấp (dưới giây), hiệu suất nhập liệu cao, khả năng thay đổi dữ liệu (mutability) và khả năng mở rộng (scalability).

### 2. KIẾN TRÚC CỐT LÕI CỦA TIMESCALEDB

TimescaleDB mở rộng PostgreSQL bằng các cơ chế kiến trúc độc đáo, được thiết kế để xử lý hiệu quả khối lượng lớn dữ liệu time-series:

#### 2.1. Hypertables (Bảng Siêu Phân Vùng)

Hypertables là khái niệm trừu tượng của bảng (table abstraction) trong TimescaleDB, tự động và minh bạch phân vùng dữ liệu theo thời gian (time) và tùy chọn theo các chiều (dimensions) khác.

- **Phân vùng tự động:** Hypertables tự động chia dữ liệu thành các phân vùng con gọi là **chunks** (khối dữ liệu), mỗi chunk chứa dữ liệu trong một khoảng thời gian cụ thể (ví dụ: mặc định là 7 ngày).
- **Hiệu suất truy vấn:** Việc phân vùng này giúp cải thiện hiệu suất truy vấn thông qua cơ chế **chunk skipping** (bỏ qua khối dữ liệu). Khi truy vấn, hệ thống chỉ quét các chunk liên quan dựa trên điều kiện `WHERE` của cột phân vùng, giảm đáng kể thời gian và tài nguyên cần thiết.
- **Quản lý dữ liệu hiệu quả:** Việc phân vùng theo thời gian giúp giữ kích thước index nhỏ, cải thiện tính cục bộ của bộ nhớ cache (cache locality), và giảm thiểu chi phí bảo trì nền (background maintenance), đảm bảo hiệu suất nhập liệu cao ngay cả khi tổng khối lượng dữ liệu tăng lên.

#### 2.2. Hypercore (Lưu Trữ Lai Hàng - Cột)

Hypercore là một công cụ lưu trữ lai (hybrid row-columnar storage engine) được thiết kế đặc biệt cho phân tích thời gian thực trên dữ liệu time-series. Nó giải quyết sự đánh đổi truyền thống giữa tốc độ ghi nhanh (row-based) và hiệu suất phân tích hiệu quả (columnar).

- **Lưu trữ theo Hàng (Row-store) cho Dữ liệu Tươi mới:** Dữ liệu mới đến (recent data) được ghi nhận ban đầu vào rowstore. Row-store được tối ưu hóa cho các thao tác `INSERT` và `UPDATE` tốc độ cao, đảm bảo các ứng dụng real-time dễ dàng xử lý các luồng dữ liệu nhanh.
- **Lưu trữ theo Cột (Column-store) cho Phân tích:** Khi dữ liệu "nguội" đi, nó được tự động chuyển đổi sang columnstore. Định dạng cột này tối ưu hóa việc quét và tổng hợp dữ liệu, cải thiện hiệu suất cho khối lượng công việc phân tích, đồng thời tiết kiệm đáng kể không gian lưu trữ (nén lên đến 98%).
- **Khả năng Thay đổi Dữ liệu (Mutability):** Hypercore hỗ trợ đầy đủ các giao dịch ACID và cho phép sửa đổi dữ liệu ngay lập tức (`INSERT`, `UPSERT`, `UPDATE`, `DELETE`), bất kể dữ liệu được lưu trữ dưới dạng hàng hay cột.

#### 2.3. Continuous Aggregates (Tổng Hợp Liên Tục)

Continuous Aggregates (CA) là các khung nhìn được hiện thực hóa (materialized views) được cập nhật một cách tăng dần (incrementally updated). Chúng là một tính năng độc quyền của TimescaleDB, hoạt động tương tự như materialized view chuẩn của Postgres nhưng được cập nhật tự động trong nền.

- **Giảm Tải Tính Toán:** CA chuyển quá trình tính toán tổng hợp từ mỗi lần truy vấn sang một bước không đồng bộ (asynchronous step) sau khi dữ liệu được nhập. Điều này giúp truy vấn đọc các kết quả đã được tính toán trước (precomputed results) thay vì quét dữ liệu thô, cải thiện đáng kể hiệu suất và hiệu quả.
- **Cập nhật Hiệu quả:** Chỉ những khoảng thời gian (time buckets) nhận dữ liệu mới hoặc bị sửa đổi mới được cập nhật, tránh việc phải tạo lại toàn bộ khung nhìn từ đầu.
- **Phân cấp Tổng hợp (Hierarchical Rollups):** Có thể tạo Continuous Aggregates dựa trên các Continuous Aggregates khác (ví dụ: tổng hợp dữ liệu theo giờ, sau đó tổng hợp tiếp thành dữ liệu theo ngày) để tinh chỉnh mức độ chi tiết của dữ liệu.
- **Time Bucketing:** CA sử dụng hàm `time_bucket` để nhóm dữ liệu thành các khoảng thời gian xác định (ví dụ: 5 phút, 1 giờ, 3 ngày).

#### 2.4. Hyperfunctions (Hàm Siêu Phân Tích)

Hyperfunctions là các hàm SQL hiệu suất cao, được xây dựng riêng cho phân tích chuỗi thời gian. Chúng hỗ trợ các phân tích phức tạp như lấp đầy khoảng trống dữ liệu (gap-filling), ước tính phân vị (percentile estimation), và trung bình trọng số thời gian (time-weighted averages).

- **Tổng hợp Bộ phận (Partial Aggregation):** Hyperfunctions cho phép TimescaleDB lưu trữ trạng thái tính toán trung gian (intermediate states) thay vì chỉ kết quả cuối cùng. Các trạng thái này có thể được hợp nhất sau này để tính toán các tổng hợp cấp cao hơn (rollups) một cách hiệu quả, giảm thiểu việc xử lý lại dữ liệu thô tốn kém.

### 3. DỰ ÁN THEO DÕI SỨC KHỎE REAL-TIME VÀ YÊU CẦU DỮ LIỆU

Dự án này tập trung vào việc xây dựng một hệ thống cho phép người dùng theo dõi các chỉ số sức khỏe cá nhân (nhịp tim, số bước chân, calo tiêu thụ) thu thập từ thiết bị đeo (wearables). Tính năng then chốt là khả năng chia sẻ dữ liệu sức khỏe theo thời gian thực (real-time) cho các thành viên trong nhóm (gia đình, bác sĩ).

Kiến trúc hệ thống được thiết kế theo mô hình bất đồng bộ, dựa trên microservice và message broker (Kafka) để đáp ứng các yêu cầu về khả năng chịu tải cao và độ trễ thấp.

#### 3.1. Vai Trò Của TimescaleDB trong Kiến Trúc

Trong kiến trúc tổng thể:

- **Lưu trữ Bền vững (Cold Path):** Dịch vụ lưu trữ (Storage Service) lắng nghe dữ liệu từ Kafka, xử lý và thực hiện `INSERT` bền vững vào **TimescaleDB** (là một phần của PostgreSQL). Mục đích là đảm bảo mọi điểm dữ liệu được lưu trữ an toàn để phân tích lịch sử.
- **Truy vấn Lịch sử (Data Retrieval):** Khi người dùng muốn xem lịch sử hoặc biểu đồ (tuần/tháng), ứng dụng gọi đến các endpoint, và dịch vụ đọc dữ liệu thực hiện các câu lệnh `SELECT` (bao gồm `JOIN` và **Continuous Aggregates**) trên PostgreSQL/TimescaleDB.

### 4. LUẬN CỨ KHOA HỌC CHO VIỆC LỰA CHỌN TIMESCALEDB

Việc lựa chọn TimescaleDB (PostgreSQL + TimescaleDB) thay vì một cơ sở dữ liệu quan hệ truyền thống hoặc NoSQL bất kỳ được căn cứ vào khả năng đáp ứng trực tiếp các yêu cầu cốt lõi của dự án Theo dõi Sức khỏe Real-time:

#### 4.1. Tối Ưu Hóa Nhập Liệu Chuỗi Thời Gian và Khả năng Mở Rộng

Dữ liệu sức khỏe từ thiết bị đeo là dữ liệu chuỗi thời gian điển hình, có tốc độ tăng trưởng nhanh (high ingest). TimescaleDB, thông qua **Hypertables**, tự động phân vùng dữ liệu theo thời gian. Điều này đảm bảo:

1.  **Hiệu suất Ghi ổn định:** Quá trình ghi (write) luôn được cô lập vào các chunk nhỏ, mới nhất, giữ cho kích thước index nhỏ và giảm thiểu tắc nghẽn, duy trì hiệu suất nhập liệu cao ngay cả khi tập dữ liệu tổng thể đạt đến hàng tỷ bản ghi.
2.  **Hỗ trợ Real-time Ingest:** **Hypercore** sử dụng row-store cho dữ liệu mới, tối ưu hóa cho `INSERT` tốc độ cao, cho phép ứng dụng xử lý nhanh chóng luồng dữ liệu đến.

#### 4.2. Khả Năng Phân Tích Lịch Sử Tốc Độ Cao

Để tạo ra các biểu đồ lịch sử theo tuần hoặc tháng (như yêu cầu trong luồng Data Retrieval), việc tổng hợp dữ liệu thô (ví dụ: tính trung bình nhịp tim theo giờ hoặc ngày) thường rất tốn kém.

1.  **Sử dụng Continuous Aggregates:** **Continuous Aggregates** giải quyết vấn đề này bằng cách tính toán trước các tổng hợp này (pre-aggregate). Khi người dùng yêu cầu xem biểu đồ hàng ngày, hệ thống truy vấn CA đã được tính toán sẵn thay vì quét toàn bộ dữ liệu thô. Điều này đảm bảo tốc độ phản hồi truy vấn dưới giây (`sub-second queries`) cho phân tích lịch sử.
2.  **Sử dụng Time Buckets:** Hàm `time_bucket` là công cụ thiết yếu để dễ dàng cuộn dữ liệu (roll up) thành các khoảng thời gian phân tích mong muốn (ví dụ: trung bình 5 phút cho cảm biến).

#### 4.3. Sự Linh Hoạt Của SQL và Khả Năng Tích Hợp

TimescaleDB được xây dựng trên PostgreSQL, cung cấp toàn bộ khả năng tương thích với SQL. Điều này cho phép dự án tận dụng các tính năng quan hệ mạnh mẽ của Postgres (như `JOIN` và các ràng buộc toàn vẹn) để xử lý logic quyền hạn và liên kết dữ liệu phức tạp của người dùng (User Profiles) với dữ liệu chuỗi thời gian (Health Metrics).

Hơn nữa, các tính năng phân tích nâng cao như **Hyperfunctions** cho phép nhóm phát triển thực hiện các tính toán chuyên sâu (ví dụ: tính toán phân vị latency, hoặc trung bình trọng số thời gian của nhịp tim) mà không cần phải xây dựng logic phức tạp bên ngoài cơ sở dữ liệu.

### 5. KẾT LUẬN

TimescaleDB không chỉ là một cơ sở dữ liệu time-series mà còn là một cơ sở dữ liệu ứng dụng hiệu suất cao, mang lại khả năng phân tích thời gian thực cho các ứng dụng. Với sự kết hợp của **Hypertables** (phân vùng tự động), **Hypercore** (lưu trữ lai), và **Continuous Aggregates** (tổng hợp gia tăng), TimescaleDB giải quyết hiệu quả thách thức kép của Dự án Theo dõi Sức khỏe Real-time: xử lý dòng dữ liệu nhập liệu tốc độ cao đồng thời cung cấp các truy vấn nhanh, có thể mở rộng trên cả dữ liệu hiện tại và lịch sử. Do đó, TimescaleDB là lựa chọn tối ưu để lưu trữ toàn bộ dữ liệu hệ thống trong dự án này.
