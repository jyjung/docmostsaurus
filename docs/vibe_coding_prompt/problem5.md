문제: 현재 파일 싱크 생성시 다음과 같이 docmost의  (workspace ? or space?) 이름의 폴더를 생성하고 이안에서 파일 작업을 바로 진행.  

jjy@jjygenmini:~/develop/docushow/docs/docmostsaurus$ tree -L 1
.
├── Security365 Common
├── Security365Common
└── SHIELD ID

이슈: 주기적으로 작업할려고 하는데 똑같은 과정으로 반복하기 때문에 이전에 생성했던 사용 안하는 폴더나 파일이 존재 
이를  임시 폴더 생성 후 교체 (Atomic Swap / Blue-Green) 로 구현 할려고 한다. 

로직:
해당하는 이름의  temp 폴더( 예 Security365 Common_temp) 폴더를 생성하고 여기에 관련 파일을 생성합니다.
생성이 완벽하게 끝나면, 기존 Security365 Common 폴더를 Security365 Common_old로 이름을 바꿉니다.

Security365 Common_temp 폴더를 Security365 Common 이름을 바꿉니다. 
Security365 Common_old를 삭제합니다.

장점:
무중단(Zero Downtime): 파일을 읽는 프로그램은 항상 완성된 폴더만 바라보게 됩니다. (생성 중에 파일이 없어서 생기는 에러 방지)
안전성: 생성 도중 에러가 나면 교체하지 않고 기존 폴더를 그대로 유지할 수 있습니다.

