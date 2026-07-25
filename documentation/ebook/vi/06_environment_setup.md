# Chương 6: Thiết lập Môi trường & Công cụ

Để chạy pipeline Kỹ thuật Harness thành công trên máy cục bộ của bạn, bạn cần cài đặt và cấu hình nền tảng công cụ cần thiết. Orchestrator (`main.go` + `internal/`) đóng vai trò là người nhạc trưởng, nhưng nó dựa vào runtime `llama.cpp` để thực hiện những công việc nặng nhọc.

## 1. Go (Golang)
Orchestrator và các module mã được tạo ra đều được viết bằng ngôn ngữ Go. Bạn sẽ cần **Go 1.26 trở lên** (xem `go.mod`).
* **Mac/Linux**:
  ```bash
  brew install go
  ```
* **Windows/Khác**: Tải xuống trình cài đặt từ [trang web chính thức của Go](https://go.dev/dl/).

## 2. Runtime Suy luận: llama.cpp
Cả ba agent (BA, Developer, DevOps) mặc định đều chạy cục bộ thông qua **llama.cpp**. Harness gọi binary `llama-completion`, vì vậy nó phải nằm trong `PATH` của bạn.
* **Mac/Linux**:
  ```bash
  brew install llama.cpp
  ```
* **Các nền tảng khác**: build từ mã nguồn theo [repository llama.cpp](https://github.com/ggml-org/llama.cpp).

Chạy cục bộ đồng nghĩa với không tốn chi phí API đám mây và đảm bảo sự riêng tư hoàn toàn cho mã nguồn của chúng ta.

## 3. Models (tự động tải xuống từ Hugging Face)
llama.cpp tự động tải xuống và lưu vào bộ nhớ đệm các model GGUF từ Hugging Face Hub bằng cách sử dụng tiền tố `hf://` — bạn không cần phải tải chúng thủ công. Các model được lưu trữ trong `~/.cache/huggingface/hub`.

Các mặc định trong `harness_config.json` là:
```json
"ba":     { "agent": "llama_cpp", "model_name": "hf://NousResearch/Hermes-3-Llama-3.1-8B-GGUF:Q4_K_M" },
"dev":    { "agent": "llama_cpp", "model_name": "hf://bartowski/Qwen2.5-Coder-14B-Instruct-GGUF:Q4_K_M" },
"devops": { "agent": "llama_cpp", "model_name": "hf://NousResearch/Hermes-3-Llama-3.1-8B-GGUF:Q4_K_M" }
```
Lần chạy đầu tiên của mỗi model sẽ tải nó xuống (việc này có thể mất một lúc); các lần chạy tiếp theo sử dụng bộ nhớ đệm. Bạn cũng có thể trỏ `model_name` tới đường dẫn của một tệp `.gguf` cục bộ thay vì một URL `hf://`.

*Tùy chọn — hoán đổi vào một CLI agent trên đám mây.* Nếu bạn muốn sử dụng một CLI agent như Claude cho một giai đoạn, hãy sao chép khối `_dev_claude_backup` không hoạt động trong `harness_config.json` đè lên khóa tương ứng và đảm bảo rằng CLI đó đã được cài đặt và xác thực. Các tích hợp MCP trong `.mcp/` (Notion, Linear) chỉ áp dụng cho các CLI agent như vậy.

## 4. Quản lý Hộp cát (Sandbox) & Kỹ năng (Skills)
Để giữ cho việc phát triển được tổ chức và duy trì các ngữ cảnh agent dạng module, pipeline hoạt động bên trong các thư mục cô lập (`workspace/`, `memory/`, `.agents/`).

Bạn có thể khởi tạo các thư mục này và chuẩn bị bộ công cụ kỹ năng cục bộ của mình bằng cách sử dụng Makefile của Harness:

### Khởi tạo Hộp cát
Đầu tiên, chạy lệnh khởi tạo để tạo các thư mục cần thiết và khởi tạo một tệp cấu hình cơ bản (baseline config):
```bash
make init
```

### Cung cấp Kỹ năng Tương tác
Tiếp theo, chạy trình cài đặt tương tác để chọn những kỹ năng tên miền chuyên gia để tải vào:
```bash
make skills
```
Kịch bản này sẽ kiểm tra Node và NPX, sau đó nhắc bạn chọn các gói chuyên gia (ví dụ: các mẫu thiết kế ClickHouse, hướng dẫn TDD, hoặc Kiến trúc Go Sạch).

Bạn cũng có thể liệt kê các kỹ năng đã cài đặt và loại bỏ chúng khi không còn cần thiết:
```bash
# Liệt kê tất cả các kỹ năng hiện đã cài đặt
make list-skills

# Loại bỏ một thư mục kỹ năng cụ thể
make remove-skill SKILL=<tên-thư-mục-kỹ-năng>
```

## 5. Xác minh Cài đặt
Khi môi trường đã được cấu hình, hãy xác minh toàn bộ thiết lập bằng một tác vụ đơn giản:
```bash
go run main.go run -task "Create a simple hello world module"
```

Nếu pipeline chạy thành công qua BA → DEV_CODING → QA_TESTING → HUMAN_IN_THE_LOOP → DEVOPS_DELIVER → MEMORY_COMPACTION → COMPLETED mà không báo bất kỳ lỗi "command not found" nào, thì môi trường của bạn đã được cấu hình hoàn hảo!
