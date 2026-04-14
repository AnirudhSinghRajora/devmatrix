# TPU Server Setup — DevMatrix / SkyWalker

Complete reference for provisioning the GCP TPU instance, installing JetStream, serving Gemma 2 9B-IT, and connecting it to the game server.

---

## Architecture

```
Internet → Caddy (HTTPS/WSS) → Go Game Server (:8080)
                                      ↓ HTTP localhost:8000
                              JetStream LLM (TPU)
                                      ↓
                           PostgreSQL (Docker, :5432)
```

The Go server calls `http://localhost:8000` directly — no network hop. JetStream runs on the same GCP instance via an attached Cloud TPU.

---

## 1. Provision the GCP TPU Instance

```bash
# Create a TPU VM (v4 or v5e recommended for Gemma 2 9B)
gcloud compute tpus tpu-vm create skywalker-tpu \
  --zone=us-central2-b \
  --accelerator-type=v4-8 \
  --version=tpu-ubuntu2204-base \
  --scopes=https://www.googleapis.com/auth/cloud-platform

# SSH into the instance
gcloud compute tpus tpu-vm ssh skywalker-tpu --zone=us-central2-b
```

> **Note:** `v4-8` gives 8 TPU cores — sufficient for Gemma 2 9B with continuous batching up to 32 concurrent requests.

---

## 2. System Dependencies

```bash
# Update packages
sudo apt-get update && sudo apt-get upgrade -y

# Python 3.10+ and pip
sudo apt-get install -y python3 python3-pip python3-venv git curl

# Create a dedicated virtualenv
python3 -m venv ~/jetstream-env
source ~/jetstream-env/bin/activate

# Upgrade pip
pip install --upgrade pip
```

---

## 3. Install JetStream

```bash
# Clone the JetStream repo
git clone https://github.com/google/JetStream.git
cd JetStream

# Install with TPU support
pip install -e ".[tpu]"

# Verify TPU devices are visible
python3 -c "import jax; print(jax.devices())"
# Expected: [TpuDevice(id=0, ...), TpuDevice(id=1, ...), ...]
```

---

## 4. Download Gemma 2 9B Weights

### Option A — Hugging Face (recommended)

```bash
pip install huggingface_hub

# Log in (requires HF account with Gemma access granted)
huggingface-cli login

# Download model weights to local disk
huggingface-cli download google/gemma-2-9b-it \
  --local-dir ~/models/gemma-2-9b-it \
  --local-dir-use-symlinks False
```

### Option B — Google Cloud Storage

```bash
# Copy from GCS bucket (if you have the weights stored there)
gsutil -m cp -r gs://your-bucket/gemma-2-9b-it ~/models/gemma-2-9b-it
```

---

## 5. Convert Weights to JetStream Format

```bash
cd ~/JetStream

python -m jetstream.tools.convert_weights \
  --model_name=gemma2-9b \
  --input_path=~/models/gemma-2-9b-it \
  --output_path=~/models/gemma-2-9b-it-jetstream
```

> This step reshards the weights for TPU memory layout. Takes ~5–10 minutes. Output lives at `~/models/gemma-2-9b-it-jetstream`.

---

## 6. Start the JetStream Inference Server

```bash
source ~/jetstream-env/bin/activate

python -m jetstream.entrypoints.http.api_server \
  --model_name=gemma2-9b \
  --tokenizer_path=~/models/gemma-2-9b-it/tokenizer.model \
  --checkpoint_path=~/models/gemma-2-9b-it-jetstream \
  --port=8000 \
  --max_batch_size=32 \
  --max_cache_length=2048
```

**Key parameters:**
| Flag | Value | Reason |
|---|---|---|
| `--max_batch_size` | `32` | At 30s cooldown with 200 players, peak is ~7 req/s — well within budget |
| `--max_cache_length` | `2048` | Our prompts are small (~800 tokens max); keep low to maximise batch slots |
| `--port` | `8000` | Must match `LLM_URL=http://localhost:8000` in the game server env |

---

## 7. Verify JetStream is Running

```bash
# Health check
curl -s http://localhost:8000/health
# Expected: {"status": "ok"} or similar

# Test a generation request
curl -s -X POST http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemma2-9b",
    "messages": [{"role": "user", "content": "Say hello in JSON"}],
    "max_tokens": 50,
    "temperature": 0.1
  }' | python3 -m json.tool
```

---

## 8. Run as a systemd Service

Create `/etc/systemd/system/jetstream.service`:

```ini
[Unit]
Description=JetStream LLM Inference Server
After=network.target

[Service]
Type=simple
User=Anirudh
WorkingDirectory=/home/Anirudh/JetStream
ExecStart=/home/Anirudh/jetstream-env/bin/python \
  -m jetstream.entrypoints.http.api_server \
  --model_name=gemma2-9b \
  --tokenizer_path=/home/Anirudh/models/gemma-2-9b-it/tokenizer.model \
  --checkpoint_path=/home/Anirudh/models/gemma-2-9b-it-jetstream \
  --port=8000 \
  --max_batch_size=32 \
  --max_cache_length=2048
Restart=on-failure
RestartSec=10s
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable jetstream
sudo systemctl start jetstream

# Check status
sudo systemctl status jetstream
sudo journalctl -u jetstream -f
```

---

## 9. Alternative: vLLM with TPU (Backup)

If JetStream has issues, vLLM provides an OpenAI-compatible API with less TPU-specific optimization:

```bash
pip install vllm[tpu]

python -m vllm.entrypoints.openai.api_server \
  --model google/gemma-2-9b-it \
  --device tpu \
  --port 8000
```

The Go server's `LLM_URL` and `LLM_MODEL` env vars don't need to change — vLLM exposes the same `/v1/chat/completions` endpoint.

---

## 10. Build & Deploy the Full Stack

### Build client

```bash
cd client
npm ci --production=false
npm run build
# Output: client/dist/
```

### Build server binary (Linux amd64)

```bash
cd server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o skywalker ./cmd/skywalker
```

### Full deploy via script

```bash
# Deploys client + server binary to the GCP instance over SSH
./deploy/deploy.sh skywalker-server
```

The script:
1. Builds the client (`npm ci && vite build`)
2. Cross-compiles the Go binary for Linux amd64
3. `scp`s the binary to `/home/Anirudh/skywalker/server/skywalker`
4. `rsync`s `client/dist/` to `/home/Anirudh/skywalker/client/dist/`
5. Installs / updates the `skywalker.service` systemd unit
6. Reloads Caddy with the latest `Caddyfile`
7. Restarts the `skywalker` service
8. Runs a health check at `http://localhost:8080/health`

---

## 11. Connect the Game Server to JetStream

Set these env vars in `/home/Anirudh/skywalker/server/.env` on the GCP instance:

```env
PORT=8080
DATABASE_URL=postgres://devmatrix_app:dev_password@localhost:5432/devmatrix?sslmode=disable
JWT_SECRET=<your-production-secret>
LLM_URL=http://localhost:8000
LLM_MODEL=gemma2-9b
LLM_WORKERS=4
PROMPT_COOLDOWN=30s
TICK_RATE=30
MAX_PLAYERS=200
ALLOWED_ORIGINS=https://skywalker.anirudhrajora.dev
```

> `LLM_URL` must be set for the server to use the real LLM. If unset, it falls back to the keyword-based mock parser automatically.

---

## 12. Start PostgreSQL (Docker)

```bash
# From the repo root on the production machine
docker compose up -d
```

Postgres starts on `:5432`. Credentials: `devmatrix_app` / `dev_password` / db `devmatrix`. DB migrations run automatically on game server startup.

---

## 13. Install & Configure Caddy

```bash
# Install Caddy on the GCP instance
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install caddy

# Deploy the project Caddyfile
sudo cp deploy/Caddyfile /etc/caddy/Caddyfile
sudo systemctl reload caddy
```

The Caddyfile proxies:
- `/ws` → `localhost:8080` (WebSocket)
- `/api/*` → `localhost:8080` (REST)
- `/health`, `/debug/*` → `localhost:8080`
- Everything else → `client/dist/` (static SPA)

---

## 14. Full Startup Order

Run these in order on the GCP instance:

```bash
# 1. Database
docker compose up -d

# 2. LLM inference server
sudo systemctl start jetstream

# 3. Game server (reads .env, runs DB migrations, connects to JetStream)
sudo systemctl start skywalker

# 4. Reverse proxy
sudo systemctl start caddy
```

---

## 15. Smoke Tests

```bash
# JetStream alive
curl -sf http://localhost:8000/health && echo "LLM OK"

# Game server alive
curl -sf http://localhost:8080/health && echo "Server OK"

# Caddy serving HTTPS
curl -sf https://skywalker.anirudhrajora.dev/health && echo "Caddy OK"

# Postgres alive
docker exec $(docker ps -qf name=postgres) pg_isready -U devmatrix_app
```

---

## 16. Troubleshooting

| Symptom | Check |
|---|---|
| JetStream won't start | `sudo journalctl -u jetstream -n 50` — often a weight path or JAX TPU init error |
| TPU not visible | `python3 -c "import jax; print(jax.devices())"` — should list TpuDevice entries |
| Game server uses mock parser | Check `LLM_URL` is set in `.env` and JetStream is responding on `:8000` |
| High LLM latency (>3s) | Reduce `--max_cache_length` or scale down to Gemma 2 2B for Tier 1–2 players |
| WebSocket disconnects | Check `ALLOWED_ORIGINS` in `.env` matches the actual frontend domain |
| DB migration fails on startup | Check `DATABASE_URL` in `.env` and that the Docker Postgres container is running |

---

## 17. Tuning for Scale

- **LLM worker count** (`LLM_WORKERS`): Start at `4`. Each worker blocks for ~0.5–2s on an HTTP call. Increase to `8` if queue depth grows under load.
- **Gemma 2B fallback**: For Tier 1–2 players, consider running a second JetStream instance on port `8001` with Gemma 2 2B-IT. Route by AI tier in `server/internal/llm/service.go`.
- **Prompt cooldown** (`PROMPT_COOLDOWN`): `30s` default keeps peak load at ~7 req/s for 200 players — well within TPU capacity.
- **Max cache length**: `2048` is generous; Tier 1 prompts are ~300 tokens. Lowering to `1024` frees TPU HBM for larger batch sizes.
