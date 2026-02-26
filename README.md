# Wiretify - Modern WireGuard VPN Management

Wiretify is a high-performance, minimalist WireGuard management tool written in Go with a sleek, modern web UI. It simplifies the setup, deployment, and administration of VPN peers, dynamic network configuration (NAT, Firewall), and IP allocation without the hassle of manual configuration files.

## 🚀 Quick Install on VPS (Recommended)

To automatically install and deploy Wiretify on your fresh VPS (Ubuntu, Debian, CentOS) in under a minute, run the following command:

```bash
wget -qO- https://raw.githubusercontent.com/accnet/Wiretify/main/deploy/install.sh | sudo bash
```

### What the installer does:
1. Installs prerequisites (`wireguard`, `iptables`, `unzip`...).
2. Enables IPv4 IP forwarding in your Linux kernel for VPN routing.
3. Detects your current Public IP automatically.
4. Downloads the newest pre-compiled release (`wiretify.zip`) directly from this repository.
5. Installs everything cleanly to `/opt/wiretify`.
6. Creates and starts a Systemd background service (`wiretify.service`) ensuring it boots up with your server.

### Post-Installation
- **Web UI Dashboard:** Access your manager at `http://<YOUR_VPS_PUBLIC_IP>:8080`
- **Firewall:** Don't forget to **open UDP port 51820** and **TCP port 8080** in your VPS's Cloud Firewall (e.g., AWS Security Group, DigitalOcean Firewall) if they are blocked.
- **Service Logs:** View realtime logs via `journalctl -fu wiretify`.

---

## 🛠️ Local Build (For Developers)

If you want to modify the source code and build Wiretify yourself:

1. Clone the repository.
2. Ensure you have `Go` and `zip` installed on your machine.
3. Run the automated build script:
   ```bash
   sudo ./deploy/build.sh
   ```
4. This script will compile the Go backend for `linux/amd64`, bundle the `web` frontend assets, and pack them neatly into `deploy/wiretify.zip` ready for deployment.

## Features
- **Zero-Config Setup:** Directly syncs with WireGuard Linux Kernel (`wgctrl`). No more messing with `wg0.conf`.
- **Exit Node Routing:** Easily specify if a device's full internet traffic should be routed through the VPN or just local traffic (Split Tunneling).
- **Auto IP Allocation:** Automatically assigns sequential IP addresses (e.g., `10.8.0.2`, `10.8.0.3`) accurately avoiding collisions.
- **Tailscale-like UI:** Beautiful, responsive, and minimalist web dashboard styled with Tailwind CSS.
