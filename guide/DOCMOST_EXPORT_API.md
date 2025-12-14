# Docmost 워크스페이스 마크다운 내보내기 API 가이드

## 개요

Docmost API를 사용하여 워크스페이스의 모든 페이지를 마크다운 형식으로 내보내는 방법을 설명합니다.

## API 엔드포인트

| 단계 | API | 메서드 | 설명 |
|------|-----|--------|------|
| 1 | `/api/auth/login` | POST | 인증 토큰 획득 |
| 2 | `/api/spaces/` | POST | 스페이스 목록 조회 |
| 3 | `/api/pages/sidebar-pages` | POST | 스페이스별 페이지 목록 조회 |
| 4 | `/api/pages/export` | POST | 개별 페이지 내보내기 |
| 5 | `/api/spaces/export` | POST | 전체 스페이스 내보내기 |

## 상세 API 사용법

### 1. 로그인 (인증 토큰 획득)

```bash
curl -c cookies.txt -X POST "http://192.168.31.101:3456/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "your-email@example.com",
    "password": "your-password"
  }'
```

**응답:**
```json
{
  "success": true,
  "status": 200
}
```

> 쿠키 파일에 `authToken`이 저장됩니다.

### 2. 스페이스 목록 조회

```bash
curl -b cookies.txt -X POST "http://192.168.31.101:3456/api/spaces/" \
  -H "Content-Type: application/json" \
  -d '{
    "limit": 100,
    "offset": 0
  }'
```

**응답:**
```json
{
  "data": {
    "items": [
      {
        "id": "019af441-79ac-7da6-9b6a-e307046aadf8",
        "name": "General",
        "description": "",
        "slug": "general",
        "visibility": "private",
        "memberCount": 1
      }
    ],
    "meta": {
      "limit": 100,
      "page": 1,
      "hasNextPage": false,
      "hasPrevPage": false
    }
  },
  "success": true,
  "status": 200
}
```

### 3. 스페이스별 페이지 목록 조회

```bash
curl -b cookies.txt -X POST "http://192.168.31.101:3456/api/pages/sidebar-pages" \
  -H "Content-Type: application/json" \
  -d '{
    "spaceId": "019af441-79ac-7da6-9b6a-e307046aadf8"
  }'
```

**응답:**
```json
{
  "data": {
    "items": [
      {
        "id": "019af441-b82b-7589-873a-f0c2de9976c8",
        "slugId": "RhJ2ushKVa",
        "title": "test",
        "icon": null,
        "parentPageId": null,
        "hasChildren": false
      }
    ],
    "meta": {
      "limit": 250,
      "page": 1,
      "hasNextPage": false
    }
  },
  "success": true,
  "status": 200
}
```

### 4. 개별 페이지 마크다운 내보내기

```bash
curl -b cookies.txt -X POST "http://192.168.31.101:3456/api/pages/export" \
  -H "Content-Type: application/json" \
  -d '{
    "pageId": "019af441-b82b-7589-873a-f0c2de9976c8",
    "format": "markdown",
    "includeAttachments": true,
    "includeChildren": false
  }' -o page_export.zip
```

**파라미터:**
| 파라미터 | 타입 | 설명 |
|----------|------|------|
| `pageId` | string | 내보낼 페이지 ID |
| `format` | string | 내보내기 형식 (`markdown` 또는 `html`) |
| `includeAttachments` | boolean | 첨부파일 포함 여부 |
| `includeChildren` | boolean | 하위 페이지 포함 여부 |

**응답:** ZIP 파일 (마크다운 파일 포함)

### 5. 전체 스페이스 마크다운 내보내기 (권장)

```bash
curl -b cookies.txt -X POST "http://192.168.31.101:3456/api/spaces/export" \
  -H "Content-Type: application/json" \
  -d '{
    "spaceId": "019af441-79ac-7da6-9b6a-e307046aadf8",
    "format": "markdown",
    "includeAttachments": true
  }' -o space_export.zip
```

**파라미터:**
| 파라미터 | 타입 | 설명 |
|----------|------|------|
| `spaceId` | string | 내보낼 스페이스 ID |
| `format` | string | 내보내기 형식 (`markdown` 또는 `html`) |
| `includeAttachments` | boolean | 첨부파일 포함 여부 |

**응답:** ZIP 파일 (스페이스 내 모든 페이지의 마크다운 파일 포함)

## 전체 워크스페이스 내보내기 스크립트

모든 스페이스를 마크다운으로 내보내는 자동화 스크립트:

```bash
#!/bin/bash

BASE_URL="http://192.168.31.101:3456/api"
EMAIL="your-email@example.com"
PASSWORD="your-password"
OUTPUT_DIR="./docmost_export"

# 출력 디렉토리 생성
mkdir -p "$OUTPUT_DIR"

# 1. 로그인
echo "로그인 중..."
curl -s -c /tmp/docmost_cookies.txt \
  -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$EMAIL\", \"password\": \"$PASSWORD\"}"

# 2. 스페이스 목록 조회
echo "스페이스 목록 조회 중..."
SPACES=$(curl -s -b /tmp/docmost_cookies.txt \
  -X POST "$BASE_URL/spaces/" \
  -H "Content-Type: application/json" \
  -d '{"limit": 100, "offset": 0}')

# 3. 각 스페이스 내보내기
echo "$SPACES" | jq -r '.data.items[] | "\(.id) \(.name)"' | while read SPACE_ID SPACE_NAME; do
  echo "스페이스 내보내기: $SPACE_NAME ($SPACE_ID)"

  curl -s -b /tmp/docmost_cookies.txt \
    -X POST "$BASE_URL/spaces/export" \
    -H "Content-Type: application/json" \
    -d "{\"spaceId\": \"$SPACE_ID\", \"format\": \"markdown\", \"includeAttachments\": true}" \
    -o "$OUTPUT_DIR/${SPACE_NAME}.zip"

  echo "저장됨: $OUTPUT_DIR/${SPACE_NAME}.zip"
done

echo "내보내기 완료!"
```

## 검증 결과

| 테스트 항목 | 결과 | 비고 |
|-------------|------|------|
| 로그인 API | ✅ 성공 | 쿠키 기반 인증 |
| 스페이스 목록 조회 | ✅ 성공 | 1개 스페이스 확인 (General) |
| 페이지 목록 조회 | ✅ 성공 | 1개 페이지 확인 (test) |
| 개별 페이지 내보내기 | ✅ 성공 | ZIP 형식, 마크다운 포함 |
| 스페이스 전체 내보내기 | ✅ 성공 | ZIP 형식, 모든 페이지 포함 |

## 내보내기 결과 예시

**General 스페이스 내보내기 (test.md):**
```markdown
# test

# 개요

md 문서를 위한 테스트 페이지 이다.
```

## 주의사항

1. **인증:** 모든 API 호출에 인증 쿠키가 필요합니다 (로그인 후 획득)
2. **응답 형식:** 내보내기 API는 ZIP 파일을 반환합니다
3. **페이지네이션:** 대량의 스페이스/페이지가 있는 경우 `limit`과 `offset` 파라미터로 페이징 처리 필요
4. **첨부파일:** `includeAttachments: true` 설정 시 첨부파일도 ZIP에 포함됩니다
