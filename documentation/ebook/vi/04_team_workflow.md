# Chương 4: Quy trình Làm việc Hàng ngày cho Kỹ sư

Bạn sẽ tương tác với Harness như thế nào hàng ngày? Bạn không cần phải là một chuyên gia về cách các vòng lặp điều phối bên trong hoạt động. Bạn chỉ cần biết cách đưa ra yêu cầu (prompt) qua giao diện dòng lệnh.

## Cấu hình

Chúng ta sử dụng kết hợp giữa `harness_config.json` và các cờ CLI để kiểm soát AI agent nào đang hoạt động.
Mặc định:
* **Phân tích Nghiệp vụ (Giai đoạn 0)** sử dụng CLI `gemini` (với khả năng Notion MCP).
* **Lập trình viên (Giai đoạn 1)** sử dụng CLI `agy` (Antigravity).
* **DevOps (Giai đoạn 3)** sử dụng một phiên bản `ollama` cục bộ (với khả năng Linear MCP).

*Lưu ý: MCP (Model Context Protocol) cho phép các AI agent tiếp cận với các công cụ bên ngoài như Notion để đọc các PRD của bạn và Linear để tự động cập nhật các thẻ công việc của bạn.*

## Chạy một Tác vụ (Task)

Nếu bạn có một tính năng cụ thể muốn hệ thống xây dựng, hãy truyền nó dưới dạng một chuỗi thô sử dụng cờ `-task`:

```bash
go run main.go -task "Tạo một hàm Fibonacci hiệu quả cao trong Go với độ phức tạp O(n)"
```

Hệ thống sẽ bắt đầu từ Giai đoạn 0, phác thảo các yêu cầu, và xây dựng mã trong thư mục `workspace/`.

## Chạy một Epic

Nếu bạn là một Giám đốc Sản phẩm (Product Manager) và có một thư mục chứa đầy các tệp yêu cầu dạng markdown thô, bạn có thể sử dụng Trình điều phối Epic:

```bash
go run main.go -epic "./requirements/v2_launch/"
```

Hệ thống sẽ phân rã tất cả các tệp trong thư mục thành các tính năng con (sub-features) tách biệt và xử lý chúng từng cái một.

## Đầu ra nằm ở đâu?

* **`memory/`**: Đây là nơi AI lưu trữ ngữ cảnh của nó. Bạn sẽ tìm thấy `definitions_of_done.md` (danh sách kiểm tra) và `system_blueprint.md` (bản đồ kiến trúc).
* **`workspace/`**: Đây là nơi mã Go thực tế được tạo ra. Mỗi tính năng sẽ có một thư mục con sạch sẽ riêng.
* **`workspace/telemetry.json`**: Kiểm tra tệp này để xem đường ống mất bao lâu, bao nhiêu dòng mã được tạo ra, và bao nhiêu lần thử tự sửa lỗi đã được sử dụng.

Chào mừng đến với tương lai của ngành kỹ thuật!
