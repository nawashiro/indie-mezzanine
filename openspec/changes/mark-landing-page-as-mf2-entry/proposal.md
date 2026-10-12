# Proposal

## Why

webmention.app は取得したHTMLに `h-entry` という文字列が含まれるとMF2として解析するが、現在のランディングページには説明文中の文字列しかなく、エントリーを検出できない。その結果、ページ内のリンクがWebmention送信候補として扱われないため、ページ自体を有効なMF2エントリーとして公開する。

## What Changes

- ランディングページを、名前と本文を持つ1件の `h-entry` としてマークアップする。
- 既存の説明内容、Markdown正本、Webmention発見情報を維持する。
- 生成HTMLに必要なMF2プロパティがあり、ページ内リンクを含む本文がエントリーの内容として解析可能であることをテストする。

## Capabilities

### New Capabilities

なし。

### Modified Capabilities

- `landing-page`: ルートページをMF2の単一 `h-entry` として公開する要件を追加する。

## Impact

- `internal/relay/landing.md` と `internal/relay/landing.go` のランディングページ生成。
- `internal/relay/landing_test.go` の生成結果テスト。
- OpenSpecの `landing-page` 仕様。
- Webmention API、投稿受付、保存形式、フィード形式、依存関係に変更はない。
