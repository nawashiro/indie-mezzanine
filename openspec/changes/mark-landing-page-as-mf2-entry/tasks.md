# Tasks

## 1. ランディングページのMF2化

- [x] 1.1 `internal/relay/landing.go` の生成HTMLで既存の主見出しを `p-name`、説明本文を `e-content`、ページ内容を単一の `h-entry` として出力し、`go test ./internal/relay -run Landing` で既存のランディングページテストが通ることを確認する。
- [x] 1.2 `internal/relay/landing_test.go` にMF2構造と名前・本文・本文内リンクの検証を追加し、対象テストで各要素を確認できることを検証する。

## 2. 統合検証

- [x] 2.1 `go test ./...` を実行し、全パッケージのテストが成功することを確認する。
