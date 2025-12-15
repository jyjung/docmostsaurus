한글 경로-romanize  동작 과정

최초 
```
├── 한글 경로 하하.md
├── 머메이드
│   └── 테스트 한글.md
├── 머메이드.md
├── _metadata.json
└── test.md
```

romanize 
```
├── hangeul-gyeongro-haha.md
├── meomeideu
│   └── teseuteu-hangeul.md
├── meomeideu.md
├── _metadata.json
└── test.md
```

romanize post processing 
같은 이름의 md 파일과 폴더가 있으면 폴더로 
```
├── hangeul-gyeongro-haha.md
├── meomeideu
│   ├── meomeideu.md
│   └── teseuteu-hangeul.md
├── _metadata.json
└── test.md
```


TODO 문제 발생 해결해야 할 사항. 

최초 
```
├── 한글 경로 하하.md
├── 머메이드
│   ├── files(아래에 md에서 사용하는 이미지들 존재)
│   └── 테스트 한글.md
├── 머메이드.md
├── _metadata.json
└── test.md
```

Problem 처리된 결과 
```
├── hangeul-gyeongro-haha.md
├── meomeideu
│   ├── meomeideu.md
│   └── teseuteu-hangeul.md
├── 머메이드
│   └── files
├── _metadata.json
└── test.md
```

원하는것. 

```
├── hangeul-gyeongro-haha.md
├── meomeideu
│   ├── files
│   ├── meomeideu.md
│   └── teseuteu-hangeul.md
├── _metadata.json
└── test.md
```

