# Chương 3: An toàn là Trên hết - Rào chắn & Tự sửa lỗi

Khi trao cho các đặc vụ tự trị khả năng viết mã, sự an toàn và độ tin cậy là ưu tiên cao nhất của chúng ta. Chúng ta đã xây dựng hai cơ chế phòng thủ chính vào Harness.

## 1. Vòng lặp Tự sửa lỗi và Ủy quyền

AI cũng mắc lỗi. Lỗi cú pháp, bài kiểm tra thất bại, và hiểu lầm yêu cầu là chuyện bình thường. Đường ống của chúng ta xử lý điều này một cách khéo léo:
* **Vòng lặp Tự sửa lỗi QA**: Nếu `go test` thất bại, đường ống sẽ nắm bắt chính xác các bản ghi biên dịch hoặc kiểm tra lỗi từ `workspace/qa_error.log` và phản hồi lại cho đặc vụ Lập trình viên. Đặc vụ có 3 nỗ lực để tự sửa mã của mình.
* **Vòng lặp Ủy quyền (Delegation Loop)**: Điều gì xảy ra nếu đặc vụ lập trình viên thất bại 3 lần? Thay vì sập hệ thống, Harness ủy quyền sự thất bại đó *ngược lại lên trên*. Nó kích hoạt giai đoạn `BA_REFACTOR`, đánh thức đặc vụ Phân tích Nghiệp vụ (BA). Đặc vụ BA sẽ phân tích các bản ghi lỗi và viết lại `definitions_of_done.md` để làm rõ các điểm mơ hồ, đảm bảo lập trình viên có cơ hội tốt hơn ở chu kỳ tiếp theo.

## 2. Cổng Quản trị & Kiểm toán Bảo mật

Trước khi bất kỳ đoạn mã do AI tạo ra nào được phép biên dịch hoặc kiểm tra, nó phải đi qua hàm `AuditGeneratedCode` bên trong `main.go`.

Hàm này phân tích tĩnh mã của AI để tìm các mẫu mã cực kỳ nguy hiểm. Nếu tìm thấy bất cứ điều gì, quá trình build sẽ thất bại ngay lập tức, và AI sẽ được hướng dẫn để loại bỏ chúng.

**Chúng ta quét tìm những gì?**
* **Thực thi Lệnh**: Chúng ta chặn gói `os/exec`. AI không được phép viết mã thực thi các lệnh shell tùy ý trên máy chủ của chúng ta.
* **Các Lệnh Phá hoại**: Các chuỗi như `rm -rf` bị nghiêm cấm.
* **Thao tác Tệp trái phép**: AI bị chặn sử dụng `os.Remove`, `os.RemoveAll`, hoặc `os.Rename` để ngăn chặn nó vô tình (hoặc cố ý) sửa đổi các tệp hệ thống bên ngoài sandbox của nó.
* **Thông tin Đăng nhập được Hardcode**: Chúng ta quét tìm `password =`, `secret =`, và `aws_access_key` để đảm bảo AI không sinh ra (hallucinate) hoặc làm rò rỉ các thông tin xác thực nhạy cảm vào mã nguồn.
