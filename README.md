# docmostsaurus

Docmost 문서를 Docusaurus 형식으로 변환하여 로컬 파일에 동기화하는 Go 프로젝트

## 개요

docmostsaurus은 Docmost의 문서를 주기적으로 가져와 Docusaurus 포맷의 마크 다운으로 변환해서 로컬 디렉토리에 저장하는 도구입니다.

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
# 이미지 빌드 및 컨테이너 실행
docker-compose up -d --build

# 또는 빌드와 실행을 분리
docker-compose build
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
| `HTTP_PORT` | HTTP 서버 포트 (헬스체크/API) | `:8080` |

> **Note**: 동시 실행 방지를 위해 `/tmp/docmostsaurus.lock` 파일을 사용합니다. 컨테이너 환경에서는 `/tmp` 디렉토리에 쓰기 권한이 필요합니다.

## 실행

### 환경변수 설정

```bash
export DOCMOST_BASE_URL="https://your-docmost-instance.com"
export DOCMOST_EMAIL="your-email@example.com"
export DOCMOST_PASSWORD="your-password"
export OUTPUT_DIR="./output"
export SYNC_INTERVAL="1h"
export HTTP_PORT=":8080"
```

### 직접 실행

```bash
go run ./cmd/docmostsaurus
```

### Docker Compose 실행

1. `.env` 파일 생성:

```bash
cp .env.example .env
# .env 파일을 편집하여 실제 값 입력
```

2. 이미지 빌드 및 실행:

```bash
docker-compose up -d --build
```

3. 로그 확인:

```bash
docker-compose logs -f docmostsaurus
```

4. 헬스체크 확인:

```bash
curl http://localhost:8080/health
```

5. 컨테이너 중지 (graceful shutdown 지원):

```bash
docker-compose down
```

## 프로젝트 구조

```
docmostsaurus/
├── cmd/
│   └── docmostsaurus/
│       └── main.go              # 엔트리포인트
├── internal/
│   ├── config/
│   │   └── config.go            # 환경변수 및 설정 관리
│   ├── docmost/
│   │   ├── client.go            # Docmost API 클라이언트 및 인증
│   │   └── export.go            # Export API 호출
│   ├── hangul/
│   │   ├── romanize.go          # 한글 로마자화 변환
│   │   └── romanize_test.go
│   ├── health/
│   │   └── health.go            # HTTP 헬스체크 서버
│   ├── lock/
│   │   └── filelock.go          # 파일 기반 동시 실행 방지
│   ├── postprocess/
│   │   ├── placeholder.go       # Placeholder/React Fragment 래핑
│   │   ├── romanize.go          # 파일명/폴더명 로마자화
│   │   ├── sanitize.go          # 특수문자 치환 및 정리
│   │   └── *_test.go            # 테스트 파일
│   └── scheduler/
│       └── scheduler.go         # 주기적 실행 스케줄러
├── docs/                        # 개발 문서
├── .env.example                 # 환경변수 예제
├── Dockerfile                   # 멀티스테이지 Docker 빌드
├── docker-compose.yml           # Docker Compose 설정
├── go.mod
├── go.sum
└── README.md
```

## 변환 결과물

Docmost에서 내보낸 마크다운을 Docusaurus에서 빌드 가능하도록 다음과 같은 후처리를 수행합니다:

### 콘텐츠 변환

| 변환 항목 | Before | After |
|----------|--------|-------|
| Placeholder 래핑 | `{variable}` | `` `{variable}` `` |
| React Fragment 래핑 | `<>`, `</>` | `` `<>` ``, `` `</>` `` |
| Raw HTML 래핑 | `<table>...</table>` | ` ```html ... ``` ` 코드블록 |

### 파일명/폴더명 변환

| 변환 항목 | Before | After |
|----------|--------|-------|
| 한글 로마자화 | `머메이드.md` | `meomeideu.md` |
| 특수문자 치환 | `C++ & Java.md` | `C-plus-plus--and--Java.md` |
| 확장자 앞 공백 제거 | `OIDC .md` | `OIDC.md` |

### 구조 변환

| 변환 항목 | 설명 |
|----------|------|
| Frontmatter 추가 | `title`, `sidebar_position` 자동 생성 |
| Slash Split 병합 | `/` 포함 제목으로 분리된 파일 병합 |
| 동명 파일/폴더 병합 | `doc.md` + `doc/` → `doc/doc.md` |
| Untitled 제거 | placeholder `untitled.md` 파일 삭제 |

### 특수문자 치환 규칙

```
& → -and-    + → -plus-    @ → -at-
# → -num-    % → -pct-     = → -eq-
(), [], {}, '', "" → 제거
```

> 상세 후처리 파이프라인은 [DOCUSAURUS_FORMAT_WORK.md](./DOCUSAURUS_FORMAT_WORK.md)를 참조하세요.

## 라이선스

MIT License
