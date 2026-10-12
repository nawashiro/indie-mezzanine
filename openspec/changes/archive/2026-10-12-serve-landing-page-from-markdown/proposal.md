# Proposal

## Why

ルートページの説明は `internal/relay/server.go` のハンドラ内に手書きHTMLとして埋め込まれており、著者と読者に必要な「投稿」「閲覧」の案内が無い。読者は公開URLに降り立っても意味が分からない。同じ案内はREADMEにあるが、READMEは運用者向けで公開URLはダミーである。説明の正本を人間が管理できるMarkdownに置き、実際の公開URLを反映したHTMLとしてルートページに出す。

## What Changes

- ルートページの説明文の正本を `landing.md` として新設し、`go:embed` で実行ファイルに同梱する。
- 起動時に一度だけ `landing.md` をHTMLへ描画してキャッシュし、ルートページはそれを返す。リクエストごとの描画コストは無い。
- `landing.md` 内の相対リンクを、設定の `PUBLIC_URL` を用いて絶対URLへ解決する。対象はWebmentionエンドポイントとcollectionの購読URL。
- ルートページの `<title>`、主見出し、`Link: <...>; rel="webmention"` ヘッダの既存挙動を維持する。
- READMEから「投稿」「閲覧」の節を `landing.md` へ移し、READMEには `landing.md` へのリンクだけを残す。二重管理を避ける。
- Markdown描画に `goldmark`（純Go）を追加する。CGOを要求せず、実行ファイルに静的にリンクする。
- 既存のWebmention受付、保存、Atom・JF2配信、ヘルス確認の挙動は変更しない。

## Capabilities

### New Capabilities

- `landing-page`: ルートページの公開説明文を、人間が管理できるMarkdownの正本から描画し、設定の公開URLを反映して配信する挙動を定義する。

### Modified Capabilities

- なし

## Impact

- コード: `internal/relay/server.go` のルートハンドラを、描画済みHTMLを返す形へ変更する。Markdown描画は新規ファイルへ分離する。
- 資産: `landing.md` を追加し、Dockerビルドへ同梱する。
- 依存: `github.com/yuin/goldmark` を追加する。純GoでCGO不要のため、「単一コンテナ」要求は不変。
- 文書: READMEの「投稿」「閲覧」を削除し、`landing.md` へのリンクを追加する。
- テスト: ルートページの本文、`PUBLIC_URL` によるリンク解決、title と主見出しの維持を検査する契約テストを追加する。
