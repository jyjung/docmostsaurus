다음과 같이 title에 / 가 있는 경우  docmost에서 잘못된 디렉토리를 구성해서 전달한다. 

_metadata.json 
```
    ...
      "id": "019a9436-1155-72e1-9c4d-e284413f5f52",
      "slugId": "joQLG26nf4",
      "title": "Planning",
      "position": "a4B57",
      "hasChildren": true,
      "children": [
            {
                "id": "019aa3f6-121b-733b-b059-5905a6628f13",
                "slugId": "TcM4lsAeEq",
                "title": "Security365 환경 인증/인가 관련 공통 에러 페이지",
                "icon": "🟢",
                "position": "a0iRl",
                "parentPageId": "019a9436-1155-72e1-9c4d-e284413f5f52",
                "hasChildren": false
            }
      ]
    ....
```

압축을 해제하면 잘못된 폴더 구성이 되어 있다. 
```
├── Outbound Provisioning.md
├── Security365 환경 인증
│   └── 인가 관련 공통 에러 페이지.md
└── Security365 계정 가입 이슈.md
```

romanize 하고난 결과는 다음과 같이 된다. 
```
├── Outbound-Provisioning.md
├── Security365-gyejeong-gaib-isyu.md
└── Security365-hwangyeong-injeung
    └── inga-gwanryeon-gongtong-ereo-peiji.md
```    

이와 같이 생성한 _metadata.json의 내용을 확인해서  title 에  "/" 가 있는 경우 정상적인 파일로 다시 합쳐야 한다. 

원하는 결과
```
├── Outbound-Provisioning.md
├── Security365-gyejeong-gaib-isyu.md
└── Security365-hwangyeong-injeung-inga-gwanryeon-gongtong-ereo-peiji.md
```    
