# docmostsaurus

Docmost 문서를 Docusaurus 형식으로 변환하여 로컬 파일에 동기화하는 Go 프로젝트

## 개요

docmostsaurus은 Docmost의 문서를 주기적으로 가져와 Docusaurus 호환 형식의 마크다운으로 변환하고, 로컬 디렉토리에 저장하는 도구입니다.

## 기능

- Docmost API를 통한 문서 자동 동기화
- Docusaurus 호환 마크다운 변환 (Frontmatter, 사이드바 구조)
- 주기적 동기화 스케줄링

## 요구사항

- Go 1.21 이상
- Docmost API 접근 권한

## 설치

### 소스에서 빌드

```bash
git clone https://github.com/jyjung/docmostsaurus.git
cd docmostsaurus
go build -o docmostsaurus ./cmd/docmostsaurus
```

### Docker 사용

```bash
docker-compose up -d
```

## 설정

환경변수를 통해 설정합니다:

| 환경변수 | 설명 | 기본값 |
|---------|------|--------|
| `DOCMOST_BASE_URL` | Docmost 서버 URL | (필수) |
| `DOCMOST_EMAIL` | Docmost 로그인 이메일 | (필수) |
| `DOCMOST_PASSWORD` | Docmost 로그인 비밀번호 | (필수) |
| `OUTPUT_DIR` | 출력 디렉토리 경로 | `./output` |
| `SYNC_INTERVAL` | 동기화 주기 (선택) | `1h` |

## 실행

### 환경변수 설정

```bash
export DOCMOST_BASE_URL="https://your-docmost-instance.com/api"
export DOCMOST_EMAIL="your-email@example.com"
export DOCMOST_PASSWORD="your-password"
export OUTPUT_DIR="./output"
```

### 직접 실행

```bash
go run cmd/docmostsaurus/main.go
```

### Docker Compose 실행

1. `.env` 파일 생성:

```bash
cp .env.example .env
# .env 파일을 편집하여 실제 값 입력
```

2. 실행:

```bash
docker-compose up -d
```

3. 로그 확인:

```bash
docker-compose logs -f
```

## 프로젝트 구조

```
docmostsaurus/
├── cmd/
│   └── docmostsaurus/
│       └── main.go           # 엔트리포인트
├── internal/
│   ├── config/
│   │   └── config.go         # 환경변수 및 설정 관리
│   ├── docmost/
│   │   ├── client.go         # Docmost API 클라이언트
│   │   ├── auth.go           # 인증 처리
│   │   └── export.go         # Export API 호출
│   ├── converter/
│   │   ├── converter.go      # 마크다운 변환 로직
│   │   ├── frontmatter.go    # Frontmatter 생성
│   │   └── sidebar.go        # 사이드바 JSON 생성
│   └── scheduler/
│       └── scheduler.go      # 주기적 실행 스케줄러
├── pkg/
│   └── markdown/
│       └── parser.go         # 마크다운 파싱 유틸리티
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## 변환 결과물

### Frontmatter

각 마크다운 파일에 Docusaurus 호환 frontmatter가 추가됩니다:

```yaml
---
id: page-unique-id
title: 페이지 제목
sidebar_label: 사이드바 표시 이름
sidebar_position: 1
description: 페이지 설명
tags:
  - tag1
  - tag2
last_update:
  date: 2024-01-01
  author: 작성자
---
```

### 사이드바 구조

Space와 Page 계층 구조를 반영한 `_category_.json` 파일이 자동 생성됩니다:

```json
{
  "label": "카테고리 이름",
  "position": 1,
  "link": {
    "type": "generated-index",
    "description": "카테고리 설명"
  }
}
```

## 라이선스

MIT License
