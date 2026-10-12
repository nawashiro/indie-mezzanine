# Tasks

## 1. feed応答のCORS許可

- [x] 1.1 `internal/relay/server.go` の `GET /collections/{name}` で、`.atom`/`.jf2` の200・404（不正UUID・未知collection・未対応接尾辞）・503の全応答に `Access-Control-Allow-Origin: *` を付し、`Access-Control-Allow-Credentials` を付けない — verify: `internal/relay/contracts_test.go` と `internal/relay/jf2_test.go` のHTTPテストに200と404のヘッダ検証（`*` がある、Credentialsが無い）を追加して通る。
- [x] 1.2 保存データ取得失敗時の503応答にも同じ許可が付く — verify: ストアを閉じてfeedを取得するHTTPテストで503とヘッダを確認する。

## 2. 統合検証

- [x] 2.1 静的検査とOpenSpec検証を実行する — verify: `go vet ./...`、`go test -race ./...`、`openspec validate allow-feed-cors --strict` が成功する。実行できない検査は未完了として報告する。

## Workflow follow-up

- 実装とレビューが済んだら、このchangeをarchiveする（`/openspec-archive-change`）。
