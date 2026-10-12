# Tasks

## 1. 依存と説明の正本

- [x] 1.1 `github.com/yuin/goldmark` を直接依存へ追加する — verify: `go mod tidy` 後に `go mod verify` が成功し、`go build ./...` が通る。
- [x] 1.2 `internal/relay/landing.md` を正本として新設する — verify: `//go:embed` で読み出せ、`## Purpose` 相当の説明と投稿・閲覧の案内を含み、H1が `indie-mezzanine` で、リンクがルート相対(`/webmention`、`/collections/{uuid}.atom` など)である。
- [x] 1.3 READMEから「投稿」「閲覧」の節を外し、`landing.md` へのリンクを残す — verify: READMEに投稿・閲覧の手順本文が残らず、`internal/relay/landing.md` へのリンクが1つ存在する。

## 2. Markdown描画と公開URL解決

- [x] 2.1 正本を起動時に一度だけHTMLへ描画して保持する処理を実装する — verify: 描画結果が非空のHTMLで、同一入出力で決定的になり、再呼び出しで再描画しないテストが成功する。
- [x] 2.2 ルート相対のリンクと画像の宛先を設定の公開URLで絶対URLへ解決する — verify: `/webmention` と `/collections/{uuid}.atom` が `PUBLIC_URL` 前置の絶対URLになり、外部の絶対URLとメールアドレスが変化しないテストが成功する。
- [x] 2.3 描画失敗を起動失敗として扱う — verify: 不正な入力で起動経路がエラーを返すテストが成功する。

## 3. ルートページの統合と文書

- [x] 3.1 ルートハンドラを、描画済みHTMLを返す形へ変更する — verify: `GET /` が `text/html; charset=utf-8` と説明本文を返し、投稿・閲覧の案内と公開URLの絶対リンクを含むHTTPテストが成功する。
- [x] 3.2 `system-identity` と発見情報の既存挙動を維持する — verify: ルートページの `<title>` と主見出しが `indie-mezzanine`、`Link: rel="webmention"` ヘッダと `<link rel="webmention">` が出力されるテストが成功する。
- [x] 3.3 既存の受付・配信テストが不変で通ることを確認する — verify: `TestExampleSubmissionAndFeed` を含む既存の契約テストが変更なしで成功する。

## 4. 統合確認

- [x] 4.1 静的検査とOpenSpec検証を実行する — verify: `go vet ./...`、`go test -race ./...`、`openspec validate serve-landing-page-from-markdown --strict` が成功する。実行できない検査は未完了として報告する。

## Workflow follow-up

- レビュー要件を満たした後に変更をアーカイブする。
- アーカイブ後、`openspec/specs/landing-page/spec.md` が作成されていることを確認する。
