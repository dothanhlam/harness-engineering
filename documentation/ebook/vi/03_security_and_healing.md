# Chương 3: An toàn là Trên hết - Rào chắn & Tự sửa lỗi

Khi trao cho các agent tự trị khả năng viết mã, sự an toàn và độ tin cậy là ưu tiên cao nhất của chúng ta. Chúng ta đã xây dựng hai cơ chế phòng thủ chính vào Harness.

## 1. Vòng lặp Tự sửa lỗi và Ủy quyền

AI cũng mắc lỗi. Lỗi cú pháp, bài kiểm tra thất bại, và hiểu lầm yêu cầu là chuyện bình thường. Pipeline của chúng ta xử lý điều này một cách khéo léo:
* **Vòng lặp Tự sửa lỗi QA**: Nếu bộ kiểm thử thất bại, pipeline sẽ nắm bắt chính xác các bản ghi lỗi biên dịch hoặc lỗi kiểm thử từ `workspace/qa_error.log` và phản hồi lại cho Developer agent. Agent có tối đa 3 nỗ lực (`MaxRetries`) để tự sửa mã của mình.
* **Vòng lặp Ủy quyền (Delegation Loop)**: Điều gì xảy ra nếu Developer agent thất bại 3 lần? Thay vì sập hệ thống, Harness ủy quyền sự thất bại đó *ngược lại lên trên*. Nó kích hoạt giai đoạn `BA_REFACTOR`, đánh thức Business Analyst agent. BA agent sẽ phân tích các bản ghi lỗi và viết lại `definitions_of_done.md` để làm rõ các điểm mơ hồ, đảm bảo lập trình viên có cơ hội tốt hơn ở chu kỳ tiếp theo.

## 2. Rào chắn Quản trị & Kiểm toán Bảo mật

Trước khi bất kỳ đoạn mã do AI tạo ra nào được phép biên dịch hoặc kiểm thử, nó phải đi qua hàm `AuditGeneratedCode` trong gói `internal/qa`. Hàm này chạy như một phần của giai đoạn `QA_TESTING`, song song với bộ kiểm thử.

Hàm này phân tích tĩnh mã của AI để tìm các mẫu mã cực kỳ nguy hiểm. Nếu tìm thấy bất cứ điều gì, quá trình build sẽ thất bại ngay lập tức, và AI sẽ được hướng dẫn để loại bỏ chúng trong vòng lặp tự sửa lỗi.

**Chúng ta quét tìm những gì?**
* **Thực thi Lệnh**: Chúng ta chặn gói `os/exec`. AI không được phép viết mã thực thi các lệnh shell tùy ý trên máy chủ của chúng ta.
* **Các Lệnh Phá hoại**: Các chuỗi như `rm -rf` bị nghiêm cấm.
* **Thao tác Tệp trái phép**: AI bị chặn sử dụng `os.Remove`, `os.RemoveAll`, hoặc `os.Rename` để ngăn chặn nó vô tình (hoặc cố ý) sửa đổi các tệp hệ thống bên ngoài sandbox của nó.
* **Thông tin Đăng nhập được Hardcode**: Chúng ta quét tìm `password =`, `secret =`, và `aws_access_key` để đảm bảo AI không sinh ra (hallucinate) hoặc làm rò rỉ các thông tin xác thực nhạy cảm vào mã nguồn.

**Các quy tắc này có thể cấu hình được.** Khi bạn chạy `harness init` trên một dự án, bộ quy tắc mặc định sẽ được ghi vào `.harness/rules.json`. Chỉnh sửa tệp đó để thêm các lệnh cấm riêng cho dự án hoặc nới lỏng các mặc định — pipeline sẽ tự động tải nó lúc chạy. Bạn cũng có thể loại trừ các module đã được tin cậy khỏi việc kiểm toán thông qua danh sách `qa_ignore` trong `harness_config.json`.
