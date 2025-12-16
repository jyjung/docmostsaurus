# MD 파일 이동 시 files 폴더 복사 처리

## 문제 상황

MD 파일을 동일한 이름의 폴더로 이동할 때, 해당 MD 파일이 참조하는 `files/` 폴더의 이미지가 함께 복사되지 않아 이미지 링크가 깨지는 문제가 있었다.

## 처리 흐름

### 1. 최초 상태 (Docmost에서 내보낸 원본)

```
├── 한글 경로 하하.md
├── 머메이드/
│   ├── files/                    ← 테스트 한글.md에서 사용하는 이미지
│   │   └── child-image.png
│   └── 테스트 한글.md
├── 머메이드.md                    ← files/ 폴더의 이미지를 참조함
├── files/                         ← 머메이드.md에서 사용하는 이미지
│   └── root-image.png
├── _metadata.json
└── test.md
```

### 2. RomanizeSpace 처리 후

한글 파일명이 로마자로 변환됨:

```
├── hangeul-gyeongro-haha.md
├── meomeideu/
│   ├── files/
│   │   └── child-image.png
│   └── teseuteu-hangeul.md
├── meomeideu.md                   ← 아직 루트에 있음
├── files/
│   └── root-image.png
├── _metadata.json
└── test.md
```

### 3. MoveFilesIntoMatchingFolders 처리 (수정된 동작)

`meomeideu.md`와 동일한 이름의 `meomeideu/` 폴더가 있으므로:

1. **files/ 폴더 복사**: 먼저 같은 레벨의 `files/` 폴더 내용을 `meomeideu/files/`로 복사
2. **MD 파일 이동**: `meomeideu.md`를 `meomeideu/meomeideu.md`로 이동

```
├── hangeul-gyeongro-haha.md
├── meomeideu/
│   ├── files/                     ← 병합됨!
│   │   ├── child-image.png        ← 기존 (teseuteu-hangeul.md용)
│   │   └── root-image.png         ← 복사됨 (meomeideu.md용)
│   ├── meomeideu.md               ← 이동됨
│   └── teseuteu-hangeul.md
├── files/                         ← 원본은 그대로 유지
│   └── root-image.png
├── _metadata.json
└── test.md
```

## 핵심 로직

### copyFilesToDestination 함수

```go
// 소스 files/ 폴더의 내용을 대상 files/ 폴더로 복사
// - 대상에 이미 같은 이름의 파일이 있으면 덮어쓰지 않음 (기존 파일 보존)
// - 하위 디렉토리도 재귀적으로 처리
func copyFilesToDestination(srcDir, dstDir string) error
```

### 처리 순서

1. `meomeideu.md`와 매칭되는 `meomeideu/` 폴더 발견
2. 같은 레벨에 `files/` 폴더가 있는지 확인
3. 있으면 `files/` 내용을 `meomeideu/files/`로 복사 (기존 파일 보존)
4. `meomeideu.md`를 `meomeideu/meomeideu.md`로 이동

## 참고

- 원본 `files/` 폴더는 삭제하지 않음 (다른 MD 파일이 참조할 수 있음)
- 대상에 동일한 파일명이 있으면 덮어쓰지 않음 (기존 파일 우선)
- 이 처리는 `MergeKoreanFoldersIntoRomanized` 전에 실행됨
