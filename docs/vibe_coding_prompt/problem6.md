현재 docmost 의 markdown export는 다음과 같이 2가지 버그가 존재한다. 

버그 1---------------------------
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

수정: 이전과 같은 로직으로  _metadata.json과 폴더구성   이를 후처리해서  _metadata.json의 title과 잘못된 폴더 구성을 수정한다. 

잘못된 폴더 구성을 수정후
```
├── Outbound Provisioning.md
├── Security365 환경 인증-인가 관련 공통 에러 페이지.md
└── Security365 계정 가입 이슈.md
```
_metadata.json도 수정사항을 반영 
```
                "title": "Security365 환경 인증-인가 관련 공통 
```


버그2--------------------------------------
버그: 이전에 삭제한 아이템이 export하는 폴더에 포함시킨다. 다만  _metadata.json에 포함되어 있지않다. 

전제조건: 버그1에 대한 처리를 먼저하고 그후에 한다. 
수정: _metadata.json 에 포함안된 파일은 체크해서 삭제한다. 