파일이름을  roman으로 변경하고 frontmatter를  추가한뒤에 
같은 meomeideu 폴더와  memeideu.md파일과 같은 레벨에서 이름이 같은 폴더와 파일이 있을때 파일을 폴더 아래로 이동시키는 로직을 작성해서 적용해줘

기존 
```
├── hangeul-gyeongro-haha.md
├── meomeideu
│   └── teseuteu-hangeul.md
├── meomeideu.md
├── _metadata.json
└── test.md
```

생성후 파일 이동
```
├── hangeul-gyeongro-haha.md
├── meomeideu
│   ├── meomeideu.md
│   └── teseuteu-hangeul.md
├── _metadata.json
└── test.md
```