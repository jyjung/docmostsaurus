# _metadata.json 생성 과정

docmostsaurus에서 `_metadata.json` 파일이 생성되는 과정을 설명합니다.

## 개요

`_metadata.json`은 Docmost에서 내보낸 문서의 계층 구조와 메타 정보를 담고 있는 파일입니다. 이 파일은 각 스페이스별로 생성되며, 페이지의 트리 구조를 완벽하게 보존합니다.

## 생성 위치

```
output/
└── {SpaceName}/
    ├── _metadata.json    ← 스페이스별 메타데이터
    ├── page1.md
    ├── page2.md
    └── ...
```

## 데이터 구조

### SpaceMeta (스페이스 메타데이터)

```go
type SpaceMeta struct {
    ID          string      // 스페이스 고유 ID
    Name        string      // 스페이스 이름
    Slug        string      // URL 슬러그
    Description string      // 스페이스 설명
    CreatedAt   string      // 생성 시간 (RFC3339)
    UpdatedAt   string      // 수정 시간 (RFC3339)
    Pages       []*PageMeta // 페이지 트리 구조
    TotalPages  int         // 전체 페이지 수
}
```

### PageMeta (페이지 메타데이터)

```go
type PageMeta struct {
    ID           string      // 페이지 UUID
    SlugID       string      // 페이지 슬러그 ID
    Title        string      // 페이지 제목
    Icon         *string     // 아이콘 (선택적)
    Position     string      // 정렬 위치 값
    ParentPageID *string     // 부모 페이지 ID
    HasChildren  bool        // 자식 페이지 존재 여부
    Children     []*PageMeta // 자식 페이지 배열
    FilePath     string      // 마크다운 파일 경로
}
```

## 생성 흐름

```
┌─────────────────────────────────────────────────────────────┐
│                      1. 인증 및 초기화                        │
│  main() → runSync() → docmost.NewClient() → Login()        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    2. 스페이스 내보내기                       │
│  ExportAllSpaces() → ExportSpace() 각 스페이스별 호출        │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ ExportSpace(space)                                   │   │
│  │  ├─ ExportSpaceAsZip(spaceID) - API에서 ZIP 다운로드 │   │
│  │  ├─ extractZip() - ZIP 파일 추출                     │   │
│  │  └─ GetSpaceMetadata(space, files) ← 메타데이터 수집 │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   3. 메타데이터 수집                         │
│  GetSpaceMetadata()                                         │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 1. ListSidebarPages(spaceID) 호출                    │   │
│  │ 2. 각 페이지에 대해 buildPageMeta() 재귀 호출         │   │
│  │    ├─ Page 정보 추출                                 │   │
│  │    ├─ 파일 경로 매칭 (findFilePathForPage)           │   │
│  │    └─ 자식 페이지 재귀 처리                           │   │
│  │ 3. sortPagesByPosition() 정렬                        │   │
│  │ 4. SpaceMeta 구조체 반환                             │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  4. _metadata.json 저장                      │
│  main.go (177-192줄)                                        │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ if exported.Metadata != nil {                        │   │
│  │     metaPath := filepath.Join(spaceDirTemp,          │   │
│  │                               "_metadata.json")      │   │
│  │     metaData, _ := json.MarshalIndent(               │   │
│  │                     exported.Metadata, "", "  ")     │   │
│  │     os.WriteFile(metaPath, metaData, 0644)           │   │
│  │ }                                                    │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     5. 포스트 프로세싱                       │
│  ├─ RemoveUntitledFiles                                    │
│  ├─ WrapPlaceholdersWithBackticks                          │
│  ├─ RomanizeSpace                                          │
│  ├─ MoveFilesIntoMatchingFolders                           │
│  ├─ MergeSlashSplitFiles                                   │
│  └─ CleanupEmptyDirs                                       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  6. 최종 배포 (Atomic Swap)                  │
│  atomicSwap() - Blue-Green 방식으로 디렉토리 교체            │
│  임시 디렉토리 → 최종 출력 디렉토리로 원자적 이동              │
└─────────────────────────────────────────────────────────────┘
```

## 핵심 코드

### 1. 메타데이터 수집 (client.go)

```go
// GetSpaceMetadata는 스페이스의 메타데이터를 수집합니다
func (c *Client) GetSpaceMetadata(space Space, files map[string][]byte) (*SpaceMeta, error) {
    // 사이드바 페이지 목록 조회
    pages, err := c.ListSidebarPages(space.ID)
    if err != nil {
        return nil, err
    }

    // 페이지 메타데이터 재귀적 빌드
    var pageMetas []*PageMeta
    for _, page := range pages {
        meta := c.buildPageMeta(page, files)
        pageMetas = append(pageMetas, meta)
    }

    // 위치순 정렬
    sortPagesByPosition(pageMetas)

    return &SpaceMeta{
        ID:          space.ID,
        Name:        space.Name,
        Slug:        space.Slug,
        Description: space.Description,
        CreatedAt:   space.CreatedAt.Format(time.RFC3339),
        UpdatedAt:   space.UpdatedAt.Format(time.RFC3339),
        Pages:       pageMetas,
        TotalPages:  countTotalPages(pageMetas),
    }, nil
}
```

### 2. 메타데이터 저장 (main.go)

```go
// _metadata.json 파일 저장
if exported.Metadata != nil {
    metaPath := filepath.Join(spaceDirTemp, "_metadata.json")
    metaData, err := json.MarshalIndent(exported.Metadata, "", "  ")
    if err != nil {
        log.Printf("Error marshaling metadata for %s: %v",
                   exported.Space.Name, err)
        processingError = err
    } else {
        if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
            log.Printf("Error writing metadata file %s: %v",
                       metaPath, err)
            processingError = err
        } else {
            log.Printf("Space '%s': metadata saved to %s",
                       exported.Space.Name, metaPath)
        }
    }
}
```

## 생성된 파일 예시

```json
{
  "id": "019a5c0a-5844-721d-87c6-046b99c11ddc",
  "name": "General",
  "slug": "general",
  "description": "일반 문서 스페이스",
  "createdAt": "2025-01-15T09:30:00Z",
  "updatedAt": "2025-01-20T14:22:33Z",
  "pages": [
    {
      "id": "019a5c34-5858-7789-8b8b-697c4f84a0c7",
      "slugId": "qkoIBs8ws5",
      "title": "시작하기",
      "position": "ZzBfg",
      "hasChildren": true,
      "children": [
        {
          "id": "019a5c34-5858-7789-8b8b-697c4f84a0c8",
          "slugId": "abc123def",
          "title": "설치 가이드",
          "position": "ZzBfh",
          "hasChildren": false,
          "children": [],
          "filePath": "시작하기/설치 가이드.md"
        }
      ],
      "filePath": "시작하기.md"
    },
    {
      "id": "019a5c34-5858-7789-8b8b-697c4f84a0c9",
      "slugId": "xyz789abc",
      "title": "API 레퍼런스",
      "position": "ZzBfk",
      "hasChildren": false,
      "children": [],
      "filePath": "API 레퍼런스.md"
    }
  ],
  "totalPages": 3
}
```

## 주요 파일 위치

| 파일 | 역할 |
|------|------|
| `internal/docmost/client.go` | SpaceMeta, PageMeta 구조체 정의 및 GetSpaceMetadata() 구현 |
| `internal/docmost/export.go` | ExportSpace(), ExportAllSpaces() 함수 |
| `cmd/docmostsaurus/main.go` | _metadata.json 저장 로직 (177-192줄) |

## 활용 용도

1. **페이지 구조 보존**: 원본 Docmost의 계층 구조를 그대로 유지
2. **파일 매핑**: 마크다운 파일과 원본 페이지 간의 관계 추적
3. **외부 도구 연동**: 다른 시스템이 페이지 구조를 재구성할 때 활용
4. **이력 관리**: 페이지 ID, 생성/수정 시간 등 메타 정보 기록
