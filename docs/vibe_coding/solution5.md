# Solution 5: Atomic Swap (Blue-Green) 구현 방안

## 문제 요약

현재 파일 싱크 시 Space 폴더에 직접 파일을 생성하기 때문에:
- 생성 중 에러 발생 시 불완전한 상태로 남음
- 이전에 생성했던 사용하지 않는 폴더/파일이 그대로 존재
- 무중단 교체 불가능

## 현재 프로그램 구조

### 전체 워크플로우

```
1. Docmost 서버 로그인
2. 모든 Space 목록 가져오기 (ListSpaces)
3. 각 Space에 대해:
   a. Space 폴더 생성 (직접 생성)
   b. ZIP 추출하여 마크다운 파일 쓰기
   c. 메타데이터 JSON 저장
   d. 12개 이상의 후처리 단계 수행
4. 완료 통계 출력
```

### 파일 구조

```
/home/jjy/develop/public/docmostsaurus/
├── cmd/docmost-file-sync/
│   ├── main.go                    # 메인 진입점 (핵심 수정 대상)
│   ├── README.md
│   └── .env
│
└── internal/
    ├── config/
    │   └── config.go              # 설정 로드
    ├── docmost/
    │   ├── client.go              # API 클라이언트
    │   └── export.go              # ZIP 추출 로직
    └── postprocess/
        ├── placeholder.go         # 플레이스홀더 처리
        ├── romanize.go            # 로마자 변환 및 폴더 생성
        └── sanitize.go            # 특수문자 정리
```

## 수정이 필요한 핵심 부분

### 1. main.go - 핵심 수정 대상

**현재 방식 (line 82-88):**
```go
spaceDir := filepath.Join(cfg.OutputDir, sanitizeDirName(exported.Space.Name))

// Create space directory
if err := os.MkdirAll(spaceDir, 0755); err != nil {
    fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", spaceDir, err)
    continue
}
```

**문제점:**
- 기존 폴더에 직접 파일을 씀
- 생성 중 에러 발생 시 불완전한 상태
- 이전 파일이 남아있음
- 무중단 교체 불가

### 2. 수정해야 할 구체적 위치

| 파일 | 라인 | 현재 동작 | 수정 필요 |
|------|------|----------|----------|
| `main.go` | 82-88 | `spaceDir` 직접 생성 | `spaceDir_temp` 생성으로 변경 |
| `main.go` | 94-105 | 파일 직접 쓰기 | temp 폴더에 쓰기 |
| `main.go` | 117-125 | 메타데이터 저장 | temp 폴더에 저장 |
| `main.go` | 127-232 | 후처리 전체 | temp 폴더에서 수행 |
| `main.go` | **신규** | 없음 | Atomic Swap 로직 추가 |

### 3. 폴더를 생성하는 함수들

**main.go:**
- `os.MkdirAll()` - line 85 (Space 폴더 생성)
- `os.MkdirAll()` - line 96 (부모 디렉토리 생성)

**romanize.go:**
- `os.MkdirAll()` - line 124 (processPage에서 로마자 폴더 생성)
- `os.MkdirAll()` - line 302 (copyFilesToDestination에서 대상 폴더 생성)

### 4. 파일을 작성하는 함수들

**main.go:**
- `os.WriteFile()` - line 102 (마크다운 파일)
- `os.WriteFile()` - line 117 (메타데이터 JSON)

**romanize.go:**
- `os.WriteFile()` - line 142 (로마자 변환된 파일)
- `os.WriteFile()` - line 334 (첨부파일 복사)

**sanitize.go:**
- `os.WriteFile()` - line 351 (병합된 파일)

**placeholder.go:**
- `os.WriteFile()` - line 36, 73, 190 (수정된 마크다운)

## 제안하는 해결 방안

### 수정된 로직 흐름

```
1. Docmost 서버 로그인
2. 모든 Space 목록 가져오기
3. 각 Space에 대해:
   a. _temp 폴더 생성 (예: "Security365 Common_temp")
   b. _temp 폴더에 모든 파일 작업 수행
   c. 모든 후처리 완료 후 Atomic Swap:
      - 기존 폴더 → _old로 이름 변경
      - _temp 폴더 → 최종 이름으로 변경
      - _old 폴더 삭제
4. 완료 통계 출력
```

### 코드 수정 예시

```go
// 현재 로직
spaceDir := filepath.Join(cfg.OutputDir, sanitizeDirName(exported.Space.Name))

// 변경할 로직
spaceName := sanitizeDirName(exported.Space.Name)
spaceDir := filepath.Join(cfg.OutputDir, spaceName)
spaceDirTemp := filepath.Join(cfg.OutputDir, spaceName+"_temp")
spaceDirOld := filepath.Join(cfg.OutputDir, spaceName+"_old")

// 1. 기존 temp 폴더가 있으면 삭제 (이전 실패한 작업 정리)
if _, err := os.Stat(spaceDirTemp); err == nil {
    os.RemoveAll(spaceDirTemp)
}

// 2. temp 폴더 생성
if err := os.MkdirAll(spaceDirTemp, 0755); err != nil {
    fmt.Fprintf(os.Stderr, "Error creating temp directory: %v\n", err)
    continue
}

// 3. 모든 파일 작업을 spaceDirTemp에서 수행
// ... (파일 쓰기, 후처리 등) ...

// 4. 성공 시 Atomic Swap 수행
if err := atomicSwap(spaceDir, spaceDirTemp, spaceDirOld); err != nil {
    fmt.Fprintf(os.Stderr, "Error during atomic swap: %v\n", err)
    // temp 폴더 정리
    os.RemoveAll(spaceDirTemp)
    continue
}
```

### Atomic Swap 함수

```go
func atomicSwap(finalDir, tempDir, oldDir string) error {
    // 1. 기존 old 폴더가 있으면 삭제
    if _, err := os.Stat(oldDir); err == nil {
        if err := os.RemoveAll(oldDir); err != nil {
            return fmt.Errorf("failed to remove old directory: %w", err)
        }
    }

    // 2. 기존 폴더가 있으면 _old로 이름 변경
    if _, err := os.Stat(finalDir); err == nil {
        if err := os.Rename(finalDir, oldDir); err != nil {
            return fmt.Errorf("failed to rename current to old: %w", err)
        }
    }

    // 3. temp 폴더를 최종 이름으로 변경
    if err := os.Rename(tempDir, finalDir); err != nil {
        // 롤백: old 폴더를 다시 원래 이름으로
        if _, statErr := os.Stat(oldDir); statErr == nil {
            os.Rename(oldDir, finalDir)
        }
        return fmt.Errorf("failed to rename temp to final: %w", err)
    }

    // 4. old 폴더 삭제
    if _, err := os.Stat(oldDir); err == nil {
        if err := os.RemoveAll(oldDir); err != nil {
            // 삭제 실패는 경고만 출력 (swap은 이미 완료됨)
            fmt.Printf("Warning: failed to remove old directory %s: %v\n", oldDir, err)
        }
    }

    return nil
}
```

## 기대 효과

### 1. 무중단 (Zero Downtime)
- 파일을 읽는 프로그램은 항상 완성된 폴더만 바라봄
- 생성 중에 파일이 없어서 생기는 에러 방지

### 2. 안전성
- 생성 도중 에러가 나면 교체하지 않고 기존 폴더를 그대로 유지
- 롤백 로직으로 실패 시 복구 가능

### 3. 깔끔한 상태 유지
- 이전에 있던 파일들이 자동으로 정리됨
- 매번 깨끗한 상태에서 시작

## 구현 시 주의사항

1. **첫 실행 시**: 기존 폴더가 없는 경우 처리 필요
2. **에러 처리**: temp 폴더 생성 후 에러 발생 시 cleanup 필요
3. **후처리 함수들**: 모든 후처리 함수에 `spaceDirTemp` 경로 전달
4. **디스크 공간**: 일시적으로 2배의 공간이 필요할 수 있음

## 다음 단계

1. `main.go`에 `atomicSwap` 함수 추가
2. Space 처리 루프에서 temp 폴더 사용하도록 수정
3. 모든 후처리 함수 호출 시 temp 경로 전달
4. 에러 발생 시 cleanup 로직 추가
5. 테스트 수행
