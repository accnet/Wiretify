# Phân tích & Kế hoạch Thực hiện: Xây dựng Wiretify (wg-easy-go)

Dựa trên tài liệu `guilde.md`, chúng tôi sẽ xây dựng một công cụ quản lý WireGuard thay thế `wg-easy` bằng Golang. Tên dự án dự kiến: **Wiretify**.

## 1. Phân tích yêu cầu bổ sung (từ `guilde2.md`)

Quy trình quản lý WireGuard trong Go không chỉ là CRUD dữ liệu mà còn là quản lý vòng đời hệ thống:
- **Runtime Initialization**: App Go phải tự động kiểm tra và khởi tạo interface `wg0` khi start.
- **Dynamic Networking**: 
    - Bật IP Forwarding (`net.ipv4.ip_forward=1`).
    - Thiết lập IP cho interface.
    - Quản lý NAT (Masquerade) và đặc biệt là **RDP Forwarding** (PREROUTING rules).
- **Direct Kernel Sync**: Nạp cấu hình từ DB thẳng vào Kernel qua Netlink/Wireguardctrl, không phụ thuộc file `.conf` vật lý.

## 2. Kiến trúc & Logic bổ sung

### Vòng đời khởi động (Startup Lifecycle)
1. **Check Dependencies**: Kiểm tra module `wireguard`.
2. **Interface Setup**: Kiểm tra/Tạo `wg0`.
3. **Network Config**: Set IP, Set Up interface, Bật IPv4 Forwarding.
4. **Firewall Setup**: Apply Postrouting (NAT) và Prerouting (RDP).
5. **Sync Peers**: Truy vấn SQLite và nạp toàn bộ Peer hiện có vào Kernel.

## 3. Kế hoạch triển khai cập nhật (Step-by-Step)

### Giai đoạn 1: Khởi tạo & Môi trường (Ngày 1)
- [ ] Khởi tạo Go mod & cấu trúc thư mục.
- [ ] **[MỚI]** Viết logic kiểm tra/khởi tạo Interface (sử dụng `netlink`).
- [ ] **[MỚI]** Viết logic thiết lập Firewall ban đầu (NAT Rules) sử dụng `go-iptables`.

### Giai đoạn 2: Database & Peer Management (Ngày 2)
- [ ] Thiết lập SQLite & GORM Models.
- [ ] Implement Service tạo Peer (Public/Private Key generation).
- [ ] **[MỚI]** Logic tự động cấp phát IP tĩnh cho Peer từ pool (ví dụ: `10.8.0.x`).

### Giai đoạn 3: WireGuard Controller Logic (Ngày 3)
- [ ] **Direct Sync**: Sử dụng `wireguardctrl` để nạp Peer vào Kernel.
- [ ] Logic "Clean start": Xóa config cũ trong Kernel và nạp mới hoàn toàn từ DB khi app khởi động.

### Giai đoạn 4: API & Web UI (Ngày 4-5)
- [ ] REST API: CRUD Peers, QR Code, Download `.conf`.
- [ ] **[MỚI]** RDP Settings API: Cấu hình port forwarding cho máy chủ đích.
- [ ] Web Dashboard: Giao diện quản lý Peers và trạng thái hệ thống.

### Giai đoạn 5: Kiểm tra & Tối ưu (Ngày 6)
- [ ] Test RDP connection thông qua VPN.
- [ ] Dockerization (yêu cầu `--cap-add=NET_ADMIN` và `--device=/dev/net/tun`).

## 4. Công nghệ sử dụng
- **Ngôn ngữ**: Go 1.21+
- **Web Framework**: Echo (nhẹ và nhanh)
- **Database**: SQLite (GORM)
- **Network**: `mdlayher/wireguardctrl`, `vishvananda/netlink`, `coreos/go-iptables`.
