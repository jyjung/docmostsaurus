# Docusaurus 포맷 호환을 위한 작업 내역

Docmost에서 내보낸 마크다운 파일을 Docusaurus에서 정상적으로 빌드할 수 있도록 수행한 후처리(post-processing) 작업들을 정리합니다.

## 목차

1. [개요](#개요)
2. [후처리 파이프라인](#후처리-파이프라인)
3. [상세 작업 내역](#상세-작업-내역)
4. [처리 순서](#처리-순서)
5. [핵심 함수 요약](#핵심-함수-요약)

---

## 개요

Docmost에서 내보낸 마크다운 파일은 Docusaurus에서 바로 사용할 수 없는 여러 문제를 포함하고 있습니다:

- 한글 파일/폴더명 (URL 호환성 문제)
- Frontmatter 누락
- `{placeholder}` 형태의 텍스트가 JSX로 해석되는 문제
- `<>`, `</>` 등 React Fragment로 해석되는 문제
- Raw HTML 테이블이 MDX에서 파싱 오류 발생
- 제목에 `/`가 포함된 경우 잘못된 폴더 구조 생성
- 특수문자가 포함된 파일명
- `untitled` placeholder 파일 생성

---

## 후처리 파이프라인

```
Docmost Export
     │
     ▼
┌─────────────────────────────────────┐
│  1. RemoveUntitledFiles             │  untitled placeholder 파일 제거
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│  2. WrapPlaceholdersWithBackticks   │  {placeholder} → `{placeholder}`
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│  3. WrapAngleBracketsWithBackticks  │  <> → `<>`, </> → `</>`
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│  4. WrapRawHTMLWithCodeBlock        │  <table>...</table> → ```html...```
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│  5. MergeSlashSplitFiles (Korean)   │  "/" 제목으로 분리된 파일 병합
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│  6. RomanizeSpace                   │  한글 → 로마자 + Frontmatter 추가
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│  7. MoveFilesIntoMatchingFolders    │  동명의 MD/폴더 병합
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│  8. MergeKoreanFoldersIntoRomanized │  한글폴더 → 로마자폴더 병합
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│  9. RenameRemainingKoreanFolders    │  남은 한글 폴더 로마자화
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│ 10. RenameRemainingKoreanFiles      │  남은 한글 MD 파일 로마자화
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│ 11. SanitizeSpecialCharacters       │  특수문자 치환 (&→-and-, 등)
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│ 12. RemoveSpaceBeforeExtension      │  "파일 .md" → "파일.md"
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│ 13. MoveFilesIntoMatchingFolders    │  (sanitization 후 재실행)
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│ 14. MergeSlashSplitFiles (Romanized)│  "/" 제목 파일 병합 (로마자화 후)
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│ 15. CleanupEmptyDirs                │  빈 디렉토리 삭제
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│ 16. RemoveUntitledFiles (Final)     │  untitled 파일 최종 제거
└─────────────────────────────────────┘
     │
     ▼
   Docusaurus Ready
```

---

## 상세 작업 내역

### 1. Placeholder 래핑 (`WrapPlaceholdersWithBackticks`)

**문제**: `{variable}` 형태의 텍스트가 MDX에서 JSX 표현식으로 해석되어 빌드 오류 발생

**해결**: 백틱으로 감싸서 코드로 표시

```
Before: 토큰 값은 {token}입니다
After:  토큰 값은 `{token}`입니다
```

**예외 처리**:
- 이미 백틱으로 감싸진 경우 스킵
- 코드 블록 내부 스킵
- 인라인 코드 내부 스킵
- 마크다운 링크 경로 내부 스킵 `[text](path)`
- JSON 패턴 스킵 `"key": {`

**파일**: `internal/postprocess/placeholder.go`

---

### 2. 빈 꺾쇠 래핑 (`WrapAngleBracketsWithBackticks`)

**문제**: `<>`, `</>` 가 React Fragment로 해석되어 빌드 오류 발생

**해결**: 백틱으로 감싸서 코드로 표시

```
Before: React의 Fragment는 <>와 </>로 표현합니다
After:  React의 Fragment는 `<>`와 `</>`로 표현합니다
```

**파일**: `internal/postprocess/placeholder.go`

---

### 3. Raw HTML 래핑 (`WrapRawHTMLWithCodeBlock`)

**문제**: `<table>`, `<tbody>`, `<tr>` 등 Raw HTML이 MDX에서 파싱 오류 발생

**해결**: HTML 코드 블록으로 래핑

```markdown
Before:
<table>
  <tr><td>데이터</td></tr>
</table>

After:
```html
<table>
  <tr><td>데이터</td></tr>
</table>
```
```

**파일**: `internal/postprocess/placeholder.go`

---

### 4. Slash Split 파일 병합 (`MergeSlashSplitFiles`)

**문제**: 제목에 `/`가 포함된 경우 Docmost가 잘못된 폴더 구조 생성

**예시**:
```
제목: "Security365 환경 인증/인가 관련 공통 에러 페이지"

잘못 생성된 구조:
└── Security365 환경 인증/
    └── 인가 관련 공통 에러 페이지.md

올바른 구조:
└── Security365-hwangyeong-injeung-inga-gwanryeon-gongtong-ereo-peiji.md
```

**처리 방식**:
- `_metadata.json`에서 `/` 포함 제목 탐색
- 한글 파일명과 로마자 파일명 모두 처리

**파일**: `internal/postprocess/sanitize.go`

---

### 5. 한글 로마자화 및 Frontmatter 추가 (`RomanizeSpace`)

**문제**: 한글 파일/폴더명이 URL 호환성 문제 발생, Frontmatter 누락

**해결**:

1. **한글 → 로마자 변환**
   ```
   머메이드.md → meomeideu.md
   테스트 한글/ → teseuteu hangeul/
   ```

2. **Frontmatter 자동 추가**
   ```yaml
   ---
   title: 머메이드
   sidebar_position: 1
   ---
   ```

**로마자화 규칙** (`internal/hangul/romanize.go`):
- 초성, 중성, 종성을 분리하여 표준 로마자로 변환
- 영문, 숫자, 공백은 그대로 유지
- 특수문자는 안전한 문자로 치환 또는 제거

```
& → -and-
+ → -plus-
@ → -at-
# → -num-
% → -pct-
= → -eq-
(), [], {}, '', "" 등 → 제거
```

**파일**: `internal/postprocess/romanize.go`, `internal/hangul/romanize.go`

---

### 6. 동명 파일/폴더 병합 (`MoveFilesIntoMatchingFolders`)

**문제**: 같은 이름의 MD 파일과 폴더가 동시에 존재

**해결**:
```
Before:
├── meomeideu.md        ← MD 파일
├── meomeideu/          ← 동명 폴더
│   └── child.md
└── files/              ← meomeideu.md가 참조하는 이미지

After:
└── meomeideu/
    ├── meomeideu.md    ← 폴더 안으로 이동
    ├── child.md
    └── files/          ← 이미지도 함께 복사
```

**파일**: `internal/postprocess/romanize.go`

---

### 7. 한글 폴더 병합 (`MergeKoreanFoldersIntoRomanized`)

**문제**: 한글 폴더와 로마자 폴더가 동시에 존재

**해결**:
```
Before:
├── 머메이드/           ← 한글 폴더
│   └── files/
└── meomeideu/          ← 로마자 폴더
    └── meomeideu.md

After:
└── meomeideu/          ← 병합된 폴더
    ├── meomeideu.md
    └── files/          ← 한글 폴더에서 병합
```

**파일**: `internal/postprocess/romanize.go`

---

### 8. 남은 한글 폴더/파일 처리

**RenameRemainingKoreanFolders**: 병합 대상이 없는 한글 폴더를 로마자로 변환

**RenameRemainingKoreanFiles**: `_metadata.json`에 없는 한글 MD 파일을 로마자로 변환

**파일**: `internal/postprocess/romanize.go`

---

### 9. 특수문자 정리 (`SanitizeSpecialCharacters`)

**문제**: 파일/폴더명에 Docusaurus가 지원하지 않는 특수문자 포함

**치환 규칙**:
```
& → -and-
+ → -plus-
@ → -at-
# → -num-
% → -pct-
= → -eq-
(), [], {}, '', "", 등 → 제거
연속 하이픈 정리: -- → -
```

**파일**: `internal/postprocess/sanitize.go`

---

### 10. 확장자 앞 공백 제거 (`RemoveSpaceBeforeExtension`)

**문제**: `"OIDC .md"`처럼 확장자 앞에 공백이 있으면 Docusaurus 청크 로딩 실패

**해결**:
```
Before: OIDC .md
After:  OIDC.md
```

**파일**: `internal/postprocess/sanitize.go`

---

### 11. Untitled 파일 제거 (`RemoveUntitledFiles`)

**문제**: Docmost가 생성한 `untitled.md`, `untitled N.md` placeholder 파일

**제거 조건**:
- 파일명: `untitled.md` 또는 `untitled N.md` (N은 숫자)
- 내용: `# untitled` 또는 `# untitled (N)`으로 시작

**파일**: `internal/postprocess/sanitize.go`

---

### 12. 빈 디렉토리 정리 (`CleanupEmptyDirs`)

**문제**: 파일 이동/삭제 후 빈 디렉토리 남음

**해결**: 비어있는 모든 디렉토리 삭제

**파일**: `internal/postprocess/romanize.go`

---

## 처리 순서

처리 순서가 중요한 이유:

1. **Placeholder/HTML 래핑은 Frontmatter 추가 전에 실행**
   - Frontmatter가 추가되면 파일 구조가 변경되어 래핑 로직이 복잡해짐

2. **Slash Split 병합은 로마자화 전/후 모두 실행**
   - 한글 파일명 상태에서 먼저 처리
   - 로마자화 후 다시 처리 (로마자화로 새로운 패턴 발생 가능)

3. **MoveFilesIntoMatchingFolders는 두 번 실행**
   - 로마자화 직후
   - SanitizeSpecialCharacters 후 (새로운 매칭 발생 가능)

4. **Untitled 제거는 처음과 끝에 실행**
   - 처음: 원본 untitled 파일 제거
   - 끝: 후처리 중 생성된 untitled 파일 제거

---

## 핵심 함수 요약

| 함수 | 위치 | 역할 |
|------|------|------|
| `WrapPlaceholdersWithBackticks` | placeholder.go | `{text}` → `` `{text}` `` |
| `WrapAngleBracketsWithBackticks` | placeholder.go | `<>` → `` `<>` `` |
| `WrapRawHTMLWithCodeBlock` | placeholder.go | HTML → ` ```html ` 블록 |
| `RomanizeSpace` | romanize.go | 한글 파일명 로마자화 + Frontmatter |
| `MoveFilesIntoMatchingFolders` | romanize.go | 동명 MD/폴더 병합 |
| `MergeKoreanFoldersIntoRomanized` | romanize.go | 한글/로마자 폴더 병합 |
| `RenameRemainingKoreanFolders` | romanize.go | 남은 한글 폴더 처리 |
| `RenameRemainingKoreanFiles` | romanize.go | 남은 한글 파일 처리 |
| `SanitizeSpecialCharacters` | sanitize.go | 특수문자 정리 |
| `MergeSlashSplitFiles` | sanitize.go | `/` 제목 파일 병합 |
| `RemoveSpaceBeforeExtension` | sanitize.go | 확장자 앞 공백 제거 |
| `RemoveUntitledFiles` | sanitize.go | untitled 파일 제거 |
| `CleanupEmptyDirs` | romanize.go | 빈 디렉토리 삭제 |
| `Romanize` | hangul/romanize.go | 한글 → 로마자 변환 |

---

## 참고

- 모든 후처리는 `cmd/docmost-file-sync/main.go`에서 순차적으로 실행
- `_metadata.json`은 Space의 페이지 계층 구조 정보를 담고 있어 Frontmatter 생성에 활용
- files/ 폴더 병합 시 동일 파일명이 있으면 기존 파일 유지 (덮어쓰지 않음)
