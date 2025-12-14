# Docmost File Sync

Docmost 워크스페이스의 모든 페이지를 마크다운 파일로 로컬에 내보내는 CLI 도구입니다.

## 기능

- Docmost의 모든 스페이스를 자동으로 조회
- 각 스페이스의 모든 페이지를 마크다운으로 내보내기
- 첨부파일 포함 지원
- 스페이스별 폴더 구조로 저장

## 설치

```bash
# 프로젝트 루트에서 빌드
go build -o docmost-file-sync ./cmd/docmost-file-sync/
```

## 환경 변수 설정

### 필수 환경 변수

| 변수 | 설명 | 예시 |
|------|------|------|
| `DOCMOST_BASE_URL` | Docmost 서버 URL | `http://192.168.31.101:3456` |
| `DOCMOST_EMAIL` | 로그인 이메일 | `user@example.com` |
| `DOCMOST_PASSWORD` | 로그인 비밀번호 | `your-password` |

### 선택 환경 변수

| 변수 | 설명 | 기본값 |
|------|------|--------|
| `OUTPUT_DIR` | 출력 디렉토리 | `./output` |

## 사용법

### 방법 1: 환경 변수 사용

```bash
export DOCMOST_BASE_URL="http://192.168.31.101:3456"
export DOCMOST_EMAIL="user@example.com"
export DOCMOST_PASSWORD="your-password"
export OUTPUT_DIR="./docmost-export"

./docmost-file-sync
```

### 방법 2: 인라인 환경 변수

```bash
DOCMOST_BASE_URL="http://192.168.31.101:3456" \
DOCMOST_EMAIL="user@example.com" \
DOCMOST_PASSWORD="your-password" \
OUTPUT_DIR="./docmost-export" \
./docmost-file-sync
```

### 방법 3: 커맨드 라인 플래그

```bash
# -output 플래그로 출력 디렉토리 지정 (환경 변수보다 우선)
DOCMOST_BASE_URL="http://192.168.31.101:3456" \
DOCMOST_EMAIL="user@example.com" \
DOCMOST_PASSWORD="your-password" \
./docmost-file-sync -output /path/to/export
```

## 출력 구조

```
output/
├── General/           # 스페이스 이름
│   ├── page1.md
│   ├── page2.md
│   └── attachments/   # 첨부파일 (있는 경우)
├── Engineering/
│   ├── docs.md
│   └── ...
└── ...
```

## 실행 예시

```
$ ./docmost-file-sync
=== Docmost Markdown Exporter ===
Server: http://192.168.31.101:3456
Output: ./output

Logging in to Docmost...
Login successful!

Exporting all spaces...
Exporting space: General (019af441-79ac-7da6-9b6a-e307046aadf8)
  Exported 5 files from space: General
Space 'General': 5 files saved to output/General

=== Export Complete ===
Total spaces: 1
Total files:  5
Output dir:   ./output
```

## 에러 처리

### 환경 변수 미설정

```
Configuration error: DOCMOST_EMAIL is required

Required environment variables:
  DOCMOST_BASE_URL  - Docmost server URL (e.g., http://192.168.31.101:3456)
  DOCMOST_EMAIL     - Docmost login email
  DOCMOST_PASSWORD  - Docmost login password

Optional environment variables:
  OUTPUT_DIR        - Output directory (default: ./output)
```

### 로그인 실패

```
Login failed: login failed with status 401: {"message":"Invalid credentials"}
```

## API 참조

이 도구는 다음 Docmost API를 사용합니다:

| API | 메서드 | 설명 |
|-----|--------|------|
| `/api/auth/login` | POST | 로그인 (쿠키 기반 인증) |
| `/api/spaces/` | POST | 스페이스 목록 조회 |
| `/api/spaces/export` | POST | 스페이스 전체 마크다운 내보내기 |

## 라이선스

MIT License
