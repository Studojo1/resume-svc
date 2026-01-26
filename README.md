# Resume Service

Minimal Go service that generates a resume PDF from JSON. No database, no auth, no job queue—just `POST /make-resume` with a resume payload.

## Overview

- **`GET /health`** — Health check.
- **`POST /make-resume`** — Request body: resume JSON. Response: `application/pdf` attachment.

Input shape: `title`, `summary`, `contact_info`, and optional `work_experiences`, `educations`, `skills`, `projects`, `certifications`. See `internal/domain/resume/model.go` for the structs.

## Requirements

- Go 1.25+

## Environment Variables

| Variable | Description |
|----------|-------------|
| `PORT`   | HTTP port (default: `8086`) |

## Build and run

```bash
go mod tidy
go build -o server ./cmd/server
./server
```

## Docker

Build from the **service directory**:

```bash
cd services/services-jobs
docker build -t resume-service .
docker run -p 8086:8086 resume-service
```

## Example

```bash
curl -X POST http://localhost:8086/make-resume \
  -H "Content-Type: application/json" \
  -d '{"title":"My Resume","summary":"...","contact_info":{"name":"Jane","email":"j@example.com"}}' \
  -o resume.pdf
```

PDF generation is currently a stub; plug in LaTeX or another renderer in `internal/tex/generator.go`.
