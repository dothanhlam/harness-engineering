# Chương 4: Quy trình Làm việc Hàng ngày cho Kỹ sư

Bạn sẽ tương tác với Harness như thế nào hàng ngày? Bạn không cần phải là một chuyên gia về cách các vòng lặp điều phối bên trong hoạt động. Bạn chỉ cần biết cách đưa ra yêu cầu (prompt) qua giao diện dòng lệnh.

## Cấu hình

Chúng ta sử dụng kết hợp giữa `harness_config.json` và các cờ CLI để kiểm soát AI agent nào đang hoạt động. Mặc định, **cả ba giai đoạn đều chạy hoàn toàn cục bộ qua llama.cpp** — không có lệnh gọi lên đám mây, đảm bảo sự riêng tư hoàn toàn cho mã nguồn của chúng ta:
* **Business Analyst (Giai đoạn 0)** sử dụng `llama_cpp` với `Hermes-3-Llama-3.1-8B`.
* **Developer (Giai đoạn 1)** sử dụng `llama_cpp` với `Qwen2.5-Coder-14B`.
* **DevOps (Giai đoạn 3)** sử dụng `llama_cpp` với `Hermes-3-Llama-3.1-8B`.

Mọi agent đều có thể hoán đổi theo từng giai đoạn. Để thử một model khác, hãy chỉnh sửa `harness_config.json` hoặc truyền một cờ (ví dụ: `-dev-model hf://bartowski/Qwen2.5-Coder-14B-Instruct-GGUF:Q4_K_M`). Bạn thậm chí có thể hoán đổi vào một CLI agent trên đám mây như Claude — repo đi kèm một khối `_dev_claude_backup` không hoạt động để làm mẫu.

*Tùy chọn: Tích hợp MCP.* Thư mục `.mcp/` giữ lại các mẫu Model Context Protocol tùy chọn (`ba_notion.json`, `devops_linear.json`) dành cho các CLI agent hỗ trợ nó — ví dụ như đọc các PRD từ Notion hoặc cập nhật các thẻ Linear. Các tích hợp này **không** được kết nối vào các agent llama.cpp cục bộ mặc định; chỉ kích hoạt chúng khi bạn chuyển sang một CLI agent nói được MCP.

## Hai lệnh con: `init` và `run`

* `harness init -project-dir .` tạo khung `.harness/rules.json` và `.agents/` vào một dự án đích (xem Chương 6 và hướng dẫn Tích hợp).
* `harness run …` thực thi pipeline.

Khi phát triển từ mã nguồn, bạn có thể chạy trực tiếp bằng `go run main.go run …` (dạng cũ `go run main.go -task …` vẫn hoạt động thông qua một cơ chế dự phòng tương thích ngược về `run`).

## Chạy một Tác vụ (Task)

Nếu bạn có một tính năng cụ thể muốn hệ thống xây dựng, hãy truyền nó dưới dạng một chuỗi thô sử dụng cờ `-task`:

```bash
go run main.go run -task "Create a highly efficient Fibonacci function in Go with O(n) complexity"
```

Hệ thống sẽ bắt đầu từ giai đoạn BA, phác thảo các yêu cầu, và xây dựng mã trong thư mục `workspace/`.

## Chạy một Epic

Nếu bạn có một thư mục chứa đầy các tệp yêu cầu dạng markdown thô, bạn có thể sử dụng Trình điều phối Epic để phân rã chúng thành các tính năng con (sub-features) tách biệt:

```bash
# Tuần tự — mỗi tính năng con chạy toàn bộ vòng lặp BA→Dev→QA→HITL, từng cái một
go run main.go run -epic "./requirements/v2_launch/"

# Song song — Dev + QA phân tán trên các tính năng con với bộ nhớ cô lập cho mỗi tác vụ
go run main.go run -epic "./requirements/v2_launch/" -parallel-epic
```

Xem **hướng dẫn Tích hợp & Epics** (`documentation/INTEGRATION.md`) để biết cách cấu trúc các tệp yêu cầu.

## Đầu ra nằm ở đâu?

* **`memory/`**: Đây là nơi AI lưu trữ ngữ cảnh của nó. Bạn sẽ tìm thấy `definitions_of_done.md` (danh sách kiểm tra) và `system_blueprint.md` (bản đồ kiến trúc).
* **`workspace/`**: Đây là nơi mã Go thực tế được tạo ra. Mỗi tính năng có một thư mục con sạch sẽ riêng, cùng với `state.json` (giai đoạn pipeline trực tiếp) và `RELEASE_NOTES.md`.
* **`telemetry.json`** (thư mục gốc của dự án): Kiểm tra tệp này để xem pipeline mất bao lâu, bao nhiêu dòng mã được tạo ra, và bao nhiêu lần thử tự sửa lỗi đã được sử dụng.

Chào mừng đến với tương lai của ngành kỹ thuật!
