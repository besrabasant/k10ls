# Kubernetes Tunnels (k10ls)

k10ls is a friendly command-line tool that keeps network tunnels open to your Kubernetes workloads. It talks directly to the Kubernetes API (no `kubectl` subprocess) so it works reliably inside scripts and automation. Point k10ls at your clusters, list the Services or Pods you care about, and it continuously maintains the port-forwards for you.

---

## Installation

### Prerequisites
- A Kubernetes cluster you can access (for example a local [kind](https://kind.sigs.k8s.io/) cluster or a managed cluster).
- Access to a kubeconfig file (usually `~/.kube/config`).
- Go 1.20 or later if you want to build from source.

### Option 1: Download a release binary
Pick the asset that matches your operating system and CPU, download it, and place it somewhere on your `PATH`.

#### Linux (x86_64)
```sh
curl -LO https://github.com/besrabasant/k10ls/releases/latest/download/k10ls-linux-amd64
chmod +x k10ls-linux-amd64
sudo mv k10ls-linux-amd64 /usr/local/bin/k10ls
```

#### macOS (Apple Silicon or Intel)
```sh
curl -LO https://github.com/besrabasant/k10ls/releases/latest/download/k10ls-darwin-universal
chmod +x k10ls-darwin-universal
sudo mv k10ls-darwin-universal /usr/local/bin/k10ls
```

#### Windows (PowerShell)
```powershell
Invoke-WebRequest -Uri "https://github.com/besrabasant/k10ls/releases/latest/download/k10ls-windows-amd64.exe" -OutFile "k10ls.exe"
# Optionally move k10ls.exe to a folder that is on your PATH.
```

### Option 2: Build from source
```sh
git clone https://github.com/besrabasant/k10ls.git
cd k10ls
make build
# The compiled binary is placed in ./bin/k10ls
```

To run the app directly from source without creating a binary:
```sh
make run
```

---

## Configuration

k10ls reads a single [TOML](https://toml.io/en/) file that describes the port-forwards you want. Save your file as `config.toml` (or any name you prefer) and pass the path with `--config`.

### Overall layout
Each configuration file has three layers:
1. **Global settings** – apply to every Kubernetes context unless overridden.
2. **Context blocks** – one block for each Kubernetes context (similar to `kubectl --context`).
3. **Resource lists** – inside a context you can list Services, Pods, or label selectors that should be forwarded.

### Configuration reference
The table below explains every option in plain English. Each row also links to a short code sample.

| Field | Scope | What it does | Example |
|-------|-------|--------------|---------|
| `global_kubeconfig` | Global | Path to a kubeconfig file used when a context does not specify its own. | [Example](#global-kubeconfig-example) |
| `default_address` | Global | IP address to bind the local ports to (use `0.0.0.0` to listen on all interfaces). | [Example](#default-address-example) |
| `[[context]]` | Per context | Starts a block that defines the Kubernetes context name and the resources you want to forward within it. | [Example](#context-block-example) |
| `name` | Context | The name of the kubeconfig context to use (matches `kubectl config get-contexts`). | [Example](#context-block-example) |
| `namespace` | Context | Default namespace for resources in this context (falls back to `default`). | [Example](#context-namespace-example) |
| `address` | Context | Overrides `default_address` for all resources in this context. | [Example](#context-address-example) |
| `kubeconfig` | Context | Path to a kubeconfig file that should be used only for this context. | [Example](#context-kubeconfig-example) |
| `[[context.svc]]` | Service | Describes a Service whose backing pods should be forwarded. | [Example](#service-forward-example) |
| `[[context.pods]]` | Pod | Describes a single Pod that should be forwarded directly. | [Example](#pod-forward-example) |
| `[[context.label-selectors]]` | Label selector | Forwards the first Pod that matches the given Kubernetes label selector. | [Example](#label-selector-example) |
| `name` | Service/Pod | Name of the Service or Pod to forward. | [Example](#service-forward-example) |
| `label` | Label selector | The label query used to find matching pods (e.g. `app=api`). | [Example](#label-selector-example) |
| `ports` | Service/Pod/Selector | List of port mappings. Each item has a `source` (local) and `target` (pod) port. | [Example](#port-mappings-example) |
| `namespace` | Resource | Optional override of the namespace for this Service/Pod/Selector. | [Example](#resource-namespace-example) |
| `address` | Resource | Optional override of the bind address for this Service/Pod/Selector. | [Example](#resource-address-example) |

### Field examples
Below are concise snippets that demonstrate how each field is used.

#### Global kubeconfig example
```toml
global_kubeconfig = "/home/alex/.kube/config"
```
This tells k10ls to load that kubeconfig file whenever a context does not specify a dedicated `kubeconfig`.

#### Default address example
```toml
default_address = "0.0.0.0"
```
All port-forwards will listen on every network interface unless a context or resource overrides the address.

#### Context block example
```toml
[[context]]
name = "kind-local"
```
Creates a new section for the `kind-local` kubeconfig context.

#### Context namespace example
```toml
[[context]]
name = "kind-local"
namespace = "demo"
```
Resources inside this context will use the `demo` namespace unless they provide their own `namespace` field.

#### Context address example
```toml
[[context]]
name = "kind-local"
address = "127.0.0.1"
```
All forwards in this context bind to `127.0.0.1`. This is useful when you only want to expose the ports on your laptop.

#### Context kubeconfig example
```toml
[[context]]
name = "prod-cluster"
kubeconfig = "/opt/configs/prod-kubeconfig"
```
Only this context uses the custom kubeconfig file. Other contexts fall back to `global_kubeconfig` or the in-cluster configuration.

#### Service forward example
```toml
[[context]]
name = "kind-local"

  [[context.svc]]
  name = "mqtt"
  ports = [
    { source = "1883", target = "1883" }
  ]
```
k10ls finds the pods behind the `mqtt` Service and forwards local port `1883` to port `1883` on that pod.

#### Pod forward example
```toml
[[context]]
name = "kind-local"

  [[context.pods]]
  name = "api-0"
  ports = [
    { source = "8080", target = "8080" }
  ]
```
Forwards directly to the Pod named `api-0`.

#### Label selector example
```toml
[[context]]
name = "kind-local"

  [[context.label-selectors]]
  label = "app=worker"
  ports = [
    { source = "9090", target = "9090" }
  ]
```
The first Pod with the label `app=worker` will be forwarded. k10ls will restart the forward if the pod changes.

#### Port mappings example
```toml
ports = [
  { source = "9000", target = "8080" },
  { source = "9001", target = "9090" }
]
```
Each mapping describes `local:remote`. The local machine listens on `9000` and sends traffic to port `8080` inside the pod, and so on.

#### Resource namespace example
```toml
[[context.svc]]
name = "dashboard"
namespace = "monitoring"
```
Overrides the context namespace when the Service lives elsewhere.

#### Resource address example
```toml
[[context.svc]]
name = "dashboard"
address = "0.0.0.0"
```
Binds this Service forward to all interfaces even if the context default is more restrictive.

---

## Examples

### Minimal configuration
```toml
# save as config.toml
global_kubeconfig = "${HOME}/.kube/config"

default_address = "127.0.0.1"

[[context]]
name = "kind-local"

  [[context.svc]]
  name = "web"
  ports = [ { source = "8080", target = "80" } ]
```
Run it with:
```sh
k10ls --config config.toml
```
You can now open `http://127.0.0.1:8080` in your browser.

### Multiple contexts and overrides
```toml
# config.multi.toml

global_kubeconfig = "${HOME}/.kube/config"
default_address = "0.0.0.0"

[[context]]
name = "kind-local"
namespace = "demo"

  [[context.svc]]
  name = "mqtt"
  ports = [
    { source = "1883", target = "1883" },
    { source = "8883", target = "8883" }
  ]

[[context]]
name = "prod"
address = "127.0.0.1"
kubeconfig = "/opt/configs/prod-kubeconfig"

  [[context.pods]]
  name = "api-0"
  namespace = "backend"
  address = "0.0.0.0"
  ports = [ { source = "9000", target = "8080" } ]

  [[context.label-selectors]]
  label = "app=worker"
  ports = [ { source = "9090", target = "9090" } ]
```
This configuration keeps tunnels open to two clusters. Notice how the prod pod overrides both the namespace and address.

---

## Usage

### Running k10ls
```sh
k10ls --config /path/to/config.toml
```
If you run without `--config`, k10ls looks for `config.toml` in the current directory.

### What you will see
When k10ls starts, it prints information about each context and forward. Here is a sample output:
```text
INFO[0000] Processing context: kind-local
INFO[0001] Started port-forward for pod web-6c6cc7cc98-zx9p7 on [8080:80]
INFO[0001] Equivalent kubectl command: kubectl --context kind-local -n default port-forward pod/web-6c6cc7cc98-zx9p7 8080:80 --address 127.0.0.1
```
- `Processing context` tells you which kubeconfig context is being processed.
- `Started port-forward` confirms the tunnels are live and shows the actual pod name.
- `Equivalent kubectl command` is a helpful hint if you want to reproduce the same port-forward manually.

k10ls keeps running until you stop it (Ctrl+C). If the pod restarts, k10ls will reconnect automatically.

### Common make commands (optional)
If you are developing k10ls, these Makefile helpers can speed things up:

- `make build` – compile the binary into `./bin/k10ls`.
- `make run` – run `go run ./...` with your current configuration.
- `make fmt` – format the Go source code.
- `make lint` – run `golangci-lint`.
- `make tidy` – clean up `go.mod` and `go.sum`.

---

## Troubleshooting tips
- **Port already in use** – choose a different `source` port or free the port with tools like `lsof -i :8080`.
- **Authentication errors** – double-check the `kubeconfig` paths and context names. Running `kubectl --context <name> get pods` is a quick sanity check.
- **No pods found** – ensure the Service has selectors or the label selector matches existing pods.

---

## License
k10ls is released under the MIT License. Contributions are welcome!
