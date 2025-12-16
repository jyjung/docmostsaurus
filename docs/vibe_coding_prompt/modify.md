# 한글 경로 Romanize 처리 과정

Docmost에서 내보낸 한글 파일/폴더명을 로마자로 변환하고, 동일한 이름의 MD 파일과 폴더를 병합하는 과정을 설명합니다.

## 전체 처리 흐름

### 1. 최초 상태 (Docmost에서 내보낸 원본)

```
├── 한글 경로 하하.md
├── 머메이드/
│   ├── files/                    ← 테스트 한글.md에서 사용하는 이미지
│   │   └── child-image.png
│   └── 테스트 한글.md
├── 머메이드.md                    ← files/ 폴더의 이미지를 참조
├── files/                         ← 머메이드.md에서 사용하는 이미지
│   └── root-image.png
├── _metadata.json
└── test.md
```

### 2. RomanizeSpace 처리

`_metadata.json`을 읽어 한글 파일명을 로마자로 변환하고 frontmatter 추가:

```
├── hangeul-gyeongro-haha.md       ← 변환됨
├── meomeideu/                      ← 폴더명 유지 (아직 한글 폴더도 존재할 수 있음)
│   ├── files/
│   │   └── child-image.png
│   └── teseuteu-hangeul.md        ← 변환됨
├── meomeideu.md                    ← 변환됨 (아직 루트에 있음)
├── files/
│   └── root-image.png
├── _metadata.json
└── test.md
```

### 3. MoveFilesIntoMatchingFolders 처리

동일한 이름의 MD 파일과 폴더가 있으면 MD 파일을 폴더 안으로 이동.
**이때 같은 레벨의 files/ 폴더 내용도 함께 복사:**

```
├── hangeul-gyeongro-haha.md
├── meomeideu/
│   ├── files/                     ← 병합됨!
│   │   ├── child-image.png        ← 기존 유지
│   │   └── root-image.png         ← 복사됨
│   ├── meomeideu.md               ← 이동됨
│   └── teseuteu-hangeul.md
├── files/                         ← 원본 유지
│   └── root-image.png
├── _metadata.json
└── test.md
```

### 4. MergeKoreanFoldersIntoRomanized 처리

한글 폴더와 로마자 폴더가 동시에 존재하면 한글 폴더 내용을 로마자 폴더로 병합:

```
예: 머메이드/ + meomeideu/ → meomeideu/ (머메이드/ 내용 병합 후 삭제)
```

### 5. RenameRemainingKoreanFolders 처리

남아있는 한글 폴더명을 로마자로 변환

### 6. RenameRemainingKoreanFiles 처리

남아있는 한글 MD 파일명을 로마자로 변환

### 7. SanitizeSpecialCharacters 처리

특수문자를 안전한 문자로 변환 (예: `&` → `-and-`)

### 8. CleanupEmptyDirs 처리

빈 디렉토리 삭제

## 최종 결과

```
├── hangeul-gyeongro-haha.md
├── meomeideu/
│   ├── files/                     ← 모든 이미지 포함
│   │   ├── child-image.png
│   │   └── root-image.png
│   ├── meomeideu.md
│   └── teseuteu-hangeul.md
├── files/                         ← 원본 유지 (다른 파일이 참조할 수 있음)
│   └── root-image.png
├── _metadata.json
└── test.md
```

## 핵심 함수

| 함수 | 역할 |
|------|------|
| `RomanizeSpace` | 한글 파일명 → 로마자 변환 + frontmatter 추가 |
| `MoveFilesIntoMatchingFolders` | 동명의 MD 파일을 폴더로 이동 + files/ 복사 |
| `MergeKoreanFoldersIntoRomanized` | 한글/로마자 폴더 병합 |
| `RenameRemainingKoreanFolders` | 남은 한글 폴더명 변환 |
| `RenameRemainingKoreanFiles` | 남은 한글 파일명 변환 |
| `SanitizeSpecialCharacters` | 특수문자 정리 |
| `CleanupEmptyDirs` | 빈 폴더 삭제 |

## 참고

- files/ 폴더 병합 시 동일 파일명이 있으면 기존 파일 유지 (덮어쓰지 않음)
- 원본 files/ 폴더는 삭제하지 않음 (다른 MD 파일이 참조할 수 있음)
