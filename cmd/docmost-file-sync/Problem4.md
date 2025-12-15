다음 경로와 같이  확장자 앞에 스페이스가 있는 파일은 Docusaurus에서 다음과 같은 오류가 발생한다.  

/home/jjy/develop/docushow/docs/docmost-sync/SHIELD ID/On-the-Job Training/jh/Authentication -and- Authorization Standards/OIDC .md

이런 확장자 앞에 스페이스가 있는 파일들은 다음과 같이 space를 붙여 주는 PostProcess를 추가해 주세요. 

/home/jjy/develop/docushow/docs/docmost-sync/SHIELD ID/On-the-Job Training/jh/Authentication -and- Authorization Standards/OIDC.md



Docusaurus 오류 ---------------

Uncaught runtime errors:
×
ERROR
Loading chunk content---docs-docmost-sync-shield-id-guide-d-5-e-d76 failed.
(error: http://localhost:3000/content---docs-docmost-sync-shield-id-guide-d-5-e-d76.js)
ChunkLoadError
    at __webpack_require__.f.j (http://localhost:3000/runtime~main.js:804:29)
    at http://localhost:3000/runtime~main.js:146:40
    at Array.reduce (<anonymous>)
    at __webpack_require__.e (http://localhost:3000/runtime~main.js:145:67)
    at fn.e (http://localhost:3000/runtime~main.js:354:50)
    at __WEBPACK_DEFAULT_EXPORT__.content---docs-docmost-sync-shield-id-guide-d-5-e-d76 (webpack-internal:///./.docusaurus/registry.js:5:56826)
    at load (webpack-internal:///./node_modules/react-loadable/lib/index.js:27:17)
    at eval (webpack-internal:///./node_modules/react-loadable/lib/index.js:55:20)
    at Array.forEach (<anonymous>)
    at loadMap (webpack-internal:///./node_modules/react-loadable/lib/index.js:54:22)