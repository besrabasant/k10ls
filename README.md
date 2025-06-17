# Kubernetes Tunnels (K10ls)

## Overview

This tool provides a **native Kubernetes API-based port-forwarding solution**, allowing users to forward ports from Kubernetes **Pods** and **Services**. Unlike `kubectl port-forward`, this tool is designed for automation, service discovery, and improved flexibility.

## Features

✅ **Forwards ports for Pods & Services**  
✅ **Resolves services to pods automatically**  
✅ **Runs natively with the Kubernetes API** (no `kubectl` subprocess)  
✅ **Supports multiple contexts & configurations**  
✅ **Written in Go** for lightweight execution  

---

## Download & Installation

### **Pre-requisites**
- **Go 1.18+**
- Kubernetes Cluster (with `kubectl` access)
- `golangci-lint` (for linting)

### **Download Binary**
#### **Linux**
```sh
curl -LO https://github.com/besrabasant/k10ls/releases/latest/download/k10ls-v1.0.0-linux-amd64
chmod +x k10ls-v1.0.0-linux-amd64
mv k10ls-v1.0.0-linux-amd64 /usr/local/bin/k10ls
```

#### **macOS**
```sh
curl -LO https://github.com/besrabasant/k10ls/releases/latest/download/k10ls-v1.0.0-macos-amd64
chmod +x k10ls-v1.0.0-macos-amd64
mv k10ls-v1.0.0-macos-amd64 /usr/local/bin/k10ls
```

#### **Windows**
```powershell
Invoke-WebRequest -Uri "https://github.com/besrabasant/k10ls/releases/latest/download/k10ls--windows-amd64.exe" -OutFile "k10ls.exe"
```

### **Build from Source**
```sh
git clone https://github.com/besrabasant/k10ls.git
cd k10ls
make build
```

### **Run the Tool**
```sh
make run
```

---

## Configuration

This tool reads configuration from a **TOML file**.

### **Example Configuration (`config.toml`)**
```toml
global_kubeconfig = "/home/user/.kube/config"
default_address = "0.0.0.0"

[[context]]
# The `name` field selects the kubeconfig context (like `kubectl --context`)
name = "kind-local"
kubeconfig = "/path/to/kubeconfig"
# address = "127.0.0.1" # optional per context

[[context.svc]]
name = "mqtt"
# address = "127.0.0.1" # optional per service
ports = [{ source = "8883", target = "8883" }, { source = "1883", target = "1883" }]

[[context.pods]]
name = "some-pod"
ports = [{ source = "8080", target = "8081" }]

[[context.label-selectors]]
label = "app=example-app"
ports = [{ source = "5000", target = "5001" }]
```

---

## Usage

### **Build & Run**
```sh
make build
make run
```

### **Available Commands**
| Command        | Description                  |
|---------------|------------------------------|
| `make build`  | Builds the application            |
| `make run`    | Runs the application         |
| `make fmt`    | Formats the Go code          |
| `make lint`   | Runs the linter (`golangci-lint`) |
| `make tidy`   | Cleans & organizes Go modules |
| `make deps`   | Installs dependencies        |
| `make clean`  | Removes build artifacts      |

---

## How It Works

1. **Reads `config.toml`** for Kubernetes contexts, services, and pods.
2. **Uses Kubernetes Go client** to interact with the cluster.
3. **Resolves services to pods** and forwards traffic dynamically.
4. **Maintains long-lived connections** with proper cleanup.

---

## Troubleshooting

### **Port Binding Issues**
If you receive:
```sh
unable to listen on any of the requested ports: [{8883 8883}]
```
It may be due to:
- Port already in use (`netstat -tulnp | grep 8883`).
- Binding restrictions (use `0.0.0.0` instead of `127.0.0.1`).

### **Debugging**
Run with logging enabled:
```sh
go run main.go --debug
```

---

## License
**MIT License** - Use freely and contribute!

---

## Contributing
1. Fork the repo
2. Create a feature branch
3. Submit a pull request

---

## Author
🚀 **Basant Besra**
📧 besrabasant@gmail.com  
🐙 [GitHub](https://github.com/besrabasant)

