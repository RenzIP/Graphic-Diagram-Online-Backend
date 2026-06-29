# GraDiOl Backend

Backend service untuk GraDiOl (Graphic Diagram Online), platform editor diagram kolaboratif berbasis web.

## Tech Stack

| Teknologi | Fungsi |
| --- | --- |
| Go 1.25 | Bahasa utama |
| Fiber v2 | HTTP framework + WebSocket |
| GORM | ORM PostgreSQL |
| Supabase/PostgreSQL | Database utama |
| Redis | Presence tracking, node locks, pub/sub |
| Supabase Auth | Autentikasi JWT |

## Setup

```bash
cp .env.example .env
go mod download
```

Isi database environment dengan connection string Supabase/PostgreSQL:

```env
SUPABASE_DATABASE_URL=postgresql://postgres:[PASSWORD]@[HOST]:5432/postgres
FRONTEND_URL=http://localhost:3000
```

## Database

Schema PostgreSQL tersedia di:

```bash
supabase/schema.sql
```

Jalankan migrasi lokal:

```bash
go run ./cmd/migrate setup
```

Seed data demo:

```bash
go run ./cmd/migrate seed
```

Reset schema:

```bash
go run ./cmd/migrate reset
```

## Development

```bash
go run ./cmd/api
```

Server berjalan di `http://localhost:8080`.

## Deploy ke Render

Blueprint deploy berada di root repo backend:

```bash
render.yaml
```

Karena backend adalah repo tersendiri, Render membangun langsung dari root repo (tidak ada `rootDir`/`buildFilter`), meng-compile dua binary (`gradiol-api` dan `gradiol-migrate`), lalu menjalankan:

```bash
./bin/gradiol-api
```

Deploy otomatis aktif (`autoDeploy: true`) — setiap push ke branch utama akan men-trigger build baru.

Env production yang perlu diisi di dashboard Render (secrets, `sync: false`):

```env
ENV=production
JWT_SECRET=...
SUPABASE_DATABASE_URL=...
FRONTEND_URL=https://domain-frontend
BACKEND_URL=https://gradiol-backend.onrender.com
REDIS_URL=...
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
GITHUB_CLIENT_ID=...
GITHUB_CLIENT_SECRET=...
```

Render otomatis mengisi `PORT`, jadi tidak perlu set manual.

Setelah deploy pertama, jalankan schema setup dari Render Shell:

```bash
./bin/gradiol-migrate setup
```

Catatan: migrasi sengaja tidak dimasukkan sebagai `preDeployCommand` karena fitur tersebut hanya tersedia untuk paid web service di Render. Untuk plan free, jalankan command migrasi secara manual dari Render Shell.

> Catatan tentang Google Cloud Functions: backend sebelumnya men-deploy ke GCF Gen 2. File entry point lama (`function.go` dan `gcf/function.go`) kini dikecualikan dari build lewat build tag `//go:build ignore`.
>
> **Langkah wajib sebelum deploy pertama** — jalankan di lokal:
>
> ```bash
> cd backend
> # 1. Hapus file entry point GCF lama (kode asli tetap tersimpan di git history):
> del function.go            # Windows (cmd)   |   rm function.go            # macOS/Linux
> rmdir /s /q gcf            # Windows (cmd)   |   rm -rf gcf                 # macOS/Linux
> # 2. Bersihkan dependency yang kini tidak terpakai:
> go mod tidy   # menghapus functions-framework-go + dependency turunannya (mis. cloudevents/sdk-go/v2)
> # 3. Pastikan build & vet bersih tanpa import GCF:
> go build ./...
> go vet ./...
> ```
>
> `go mod tidy` bersifat **wajib**, bukan opsional: tanpa itu, `functions-framework-go` tetap tercatat sebagai direct dependency di `go.mod` dan dapat membuat gate `go mod tidy --check`/`go mod verify` di CI gagal. Jika Anda lebih suka tidak menghapus file GCF, biarkan saja — keduanya sudah diberi build tag `//go:build ignore` sehingga tidak ikut dikompilasi. Tapi menghapus lebih bersih.

## API

| Method | Endpoint | Deskripsi |
| --- | --- | --- |
| GET | `/api/health` | Health check |
| POST | `/api/auth/callback` | OAuth callback handler |
| GET | `/api/workspaces` | List user workspaces |
| POST | `/api/workspaces` | Create workspace |
| PUT | `/api/workspaces/:id` | Update workspace |
| DELETE | `/api/workspaces/:id` | Delete workspace |
| GET | `/api/workspaces/:id/projects` | List projects in workspace |
| POST | `/api/projects` | Create project |
| PUT | `/api/projects/:id` | Update project |
| DELETE | `/api/projects/:id` | Delete project |
| GET | `/api/projects/:id/documents` | List documents in project |
| POST | `/api/documents` | Create document |
| GET | `/api/documents/:id` | Get document detail |
| PUT | `/api/documents/:id` | Update document |
| DELETE | `/api/documents/:id` | Delete document |

## WebSocket

```text
ws://localhost:8080/ws/:documentId
```
