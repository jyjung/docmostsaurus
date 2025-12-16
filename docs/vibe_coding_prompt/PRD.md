# doc2git

Docmost 문서를 Docusaurus 형식으로 변환하여 Git 저장소에 동기화하는 Go 프로젝트

## 개요

doc2git은 Docmost의 문서를 주기적으로 가져와 Docusaurus 호환 형식의 마크다운으로 변환하고, Git 저장소에 자동으로 커밋하는 도구입니다.

## 기술 스택

- **언어**: Go
- **대상 소스**: Docmost (문서 관리 플랫폼)
- **출력 형식**: Docusaurus 호환 마크다운

## 주요 기능

### 1. 인증 및 설정

- 환경변수를 통한 Docmost 인증 정보 관리
  - `DOCMOST_CLIENT_ID`: Docmost API 클라이언트 ID
  - `DOCMOST_CLIENT_SECRET`: Docmost API 클라이언트 시크릿
  - `DOCMOST_BASE_URL`: Docmost 서버 URL
- 추가 설정 옵션
  - `SYNC_INTERVAL`: 동기화 주기 (기본값: 1시간)
  - `OUTPUT_DIR`: 출력 디렉토리 경로
  - `GIT_REPO_PATH`: Git 저장소 경로
  - `GIT_BRANCH`: 커밋할 브랜치 (기본값: main)

### 2. 문서 동기화

- 지정된 주기(기본 1시간)마다 Docmost에서 문서 동기화
- Docmost Export Page API를 활용하여 접근 가능한 Space의 모든 Page를 마크다운 형식으로 다운로드
- 변경된 문서만 선별적으로 처리 (incremental sync)

### 3. Docusaurus 형식 변환

#### 3.1 Frontmatter 추가

각 마크다운 파일에 Docusaurus 호환 frontmatter 메타데이터 추가:

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

#### 3.2 사이드바 구조 생성

Space와 Page 계층 구조를 반영한 `_category_.json` 파일 자동 생성:

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

#### 3.3 컨텐츠 변환

- Docmost 내부 링크를 Docusaurus 상대 경로로 변환
- 이미지 경로 변환 및 static 디렉토리로 복사
- 코드 블록 언어 태그 정규화
- Admonition(경고, 정보 박스) 문법 변환

### 4. Git 연동

- 변환 완료 후 자동 Git 커밋
- 커밋 메시지에 동기화 시간 및 변경된 문서 목록 포함
- 선택적 자동 push 기능

## 디렉토리 구조

```
doc2git/
├── cmd/
│   └── doc2git/
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
│   ├── git/
│   │   └── git.go            # Git 작업 처리
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

## 실행 방법

### 환경변수 설정

```bash
export DOCMOST_CLIENT_ID="your-client-id"
export DOCMOST_CLIENT_SECRET="your-client-secret"
export DOCMOST_BASE_URL="https://your-docmost-instance.com"
export OUTPUT_DIR="./output"
export GIT_REPO_PATH="./docusaurus-docs"
```

### 직접 실행

```bash
go run cmd/doc2git/main.go
```

### Docker 실행

```bash
docker-compose up -d
```

## 워크플로우

```
┌─────────────────┐
│   Scheduler     │
│  (1시간 주기)   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Docmost API    │
│  인증 및 연결   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Export Pages   │
│  (마크다운)     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Converter     │
│ - Frontmatter   │
│ - Sidebar JSON  │
│ - 링크 변환     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Git Commit    │
│   (자동 커밋)   │
└─────────────────┘
```

## 향후 확장 계획

- [ ] Webhook 지원으로 실시간 동기화
- [ ] 다중 Space 선택적 동기화
- [ ] 충돌 감지 및 해결 메커니즘
- [ ] 웹 대시보드 UI
- [ ] Slack/Discord 알림 연동
