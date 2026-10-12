# Design

## Context

動機は `proposal.md` を参照する。

現状、ルートページの本文は `internal/relay/server.go` のハンドラ内に `fmt.Fprintf` の手書きHTMLとして埋め込まれている。同じ場所で `Link: rel="webmention"` ヘッダと `<link rel="webmention">` を出しており、これは取得側への発見情報として機能している。`system-identity` はルートページの `<title>` と主見出しを `indie-mezzanine` とすることを要求する。`single-container-runtime` はリレー本体を単一コンテナに収め、外部のDB・Redis・SMTPを要求しないことを条件とする。

リポジトリには実行ファイルへ埋め込まれた資産はまだ無く、`examples/post.html` はテストが `os.ReadFile` で読むだけである。

## Goals / Non-Goals

**Goals:**
- ルートページの説明を、コードを触らず編集できる単一のMarkdown正本から生成する。
- 表示は実際の公開URLを反映し、`system-identity` と `Link rel="webmention"` の既存挙動を維持する。
- 変換コストをリクエスト経路から排除し、起動時のみに閉じ込める。

**Non-Goals:**
- Markdown以外の記法やテンプレートエンジンの導入。
- ルートページ以外のページ、ナビゲーション、CSSや画像の同梱。
- 既存の受付・保存・Atom/JF2配信・ヘルス確認の変更。
- Markdownの描画結果に対するリッチなHTML制御(サニタイズ範囲はデフォルトのgoldmark出力に限る)。

## Decisions

### 正本の場所と同梱方法

`internal/relay/landing.md` を正本とし、`internal/relay` パッケージで `//go:embed` する。

- `go:embed` はパッケージ相対で `..` を辿れない。正本をコードと同じパッケージに置けば、埋め込みが1ディレクティブで済む。
- ルート直下や `docs/` に置くと、埋め込みのために専用パッケージを足す必要があり、構成が増える。READMEからは相対パスでリンクすれば発見性は保てる。
- Dockerビルドは既に `COPY internal ./internal` を行うため、正本の追加に `Dockerfile` の変更は不要。

代替案: ルート直下に置く(発見性は高いが埋め込み用パッケージが必要)、ハンドラ内に文字列で持つ(人間が管理できないので却下)。

### Markdown描画の時期とライブラリ

`github.com/yuin/goldmark` を追加し、起動時に一度だけHTMLへ描画してキャッシュする。

- 純正GoにMarkdownパーサは無い。goldmarkは純Go・CGO不要で、実行ファイルへ静的にリンクする。`single-container-runtime` の「外部サービスを要求しない」「非root」を損なわない。
- 描画をリクエスト時にやると無駄なコストになる。`NewApp` 系の起動経路で1回描画し、失敗時は起動失敗として早期に落とす。
- 代替案: ビルド時に外部ツールでHTML化(ビルドパイプラインに外部依存が増え、単一バイナリ思想に反する)、手書きHTML(却下)。

### 公開URLの解決

`landing.md` はアプリのルート相対パス(`/webmention`、`/collections/{uuid}.atom` など)でリンクを書く。描画時に、リンクと画像の宛先がルート相対なら `PUBLIC_URL` を前置して絶対URLへ解決する。

- 正本はMarkdownとして自然なまま保てる。`{{PUBLIC_URL}}` のようなトークンやフルURLの直書きは、正本を読みにくくする。
- 実装はMarkdownをASTへ解析し、リンク・画像ノードの宛先を書き換えてから描画する。goldmarkのASTは走査できる。
- 外部の絶対URLとメールアドレスはそのまま残す。

代替案: 正本へフルURLを書く(公開URL変更に追従できない)、HTML描画後に文字列置換する(誤置換の危険)。

### 既存挙動の維持

ルートハンドラは、描画済みHTMLを `text/html; charset=utf-8` で返す以外の構造を変えない。`<title>` と主見出しは `indie-mezzanine` のまま(正本のH1を `indie-mezzanine` に固定)、`Link: rel="webmention"` ヘッダと `<link rel="webmention">` は現行どおり出力する。他のハンドラ、保存モデル、Atom/JF2生成は触らない。

### READMEの再構成

READMEから「投稿」「閲覧」の節を `landing.md` へ移し、READMEには `landing.md` へのリンクを残す。同じ文章を二箇所に置かない。`examples/post.html` はマークアップの実例として残し、`landing.md` から参照する。

## Risks / Trade-offs

- [goldmarkが直接依存と実行ファイルサイズを増やす] → 純Go・CGO不要で単一バイナリ条件は不変。許容する。
- [正本のH1を編集すると `system-identity` を破る] → 契約テストで `<title>` と主見出しが `indie-mezzanine` であることを検査する。
- [ルート相対の解決漏れ(画像・アンカー)] → リンクと画像の宛先を対象に限定し、テストで絶対URL化を検査する。
- [投稿案内が `landing.md` と `examples/post.html` に分散する] → `landing.md` から実例へリンクし、正本を一本化する。
- [公開URL未設定時の解決] → 既存の `Config.Validate` が公開URLを必須とする前提を維持し、解決はその値に依存する。

## Migration Plan

差分は表示本文の生成経路のみで、保存データへ触れない。配備はイメージ差し替えで完了し、ロールバックも旧イメージへ戻すだけ。既存の受付・配信・保存はそのまま動く。
