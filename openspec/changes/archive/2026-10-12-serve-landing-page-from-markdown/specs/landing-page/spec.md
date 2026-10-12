# Spec Delta

## Purpose

ルートページを、著者と読者に向けた公開説明の窓口とする。サービスの意味と、投稿・閲覧の方法を人間が管理できる正本から描画し、実際の公開URLを反映して配信する。

## ADDED Requirements

### Requirement: ルートページのサービス説明

ルートページは、HTMLでサービスの説明を提供する SHALL。説明は著者に向けた投稿の案内と、読者に向けた閲覧の案内を含む SHALL。

#### Scenario: 投稿案内の表示
- **WHEN** 著者がルートページを取得する
- **THEN** システムは公開ページのマークアップと、UUID collection への `rel="collection"` リンクと、Webmention の送信先を説明する

#### Scenario: 閲覧案内の表示
- **WHEN** 読者がルートページを取得する
- **THEN** システムは collection 単位の Atom と JF2 Feed の購読方法を説明する

### Requirement: 公開URLの反映

ルートページが示す Webmention エンドポイントと購読URLは、設定の公開URLで解決した絶対URLとする SHALL。ルートページはダミーのホスト名を含まない MUST NOT。

#### Scenario: 設定済み公開URLの反映
- **WHEN** 運用者が公開URLを設定してルートページを取得する
- **THEN** Webmention エンドポイントと購読URLは、その公開URLを起点とした絶対URLになる

### Requirement: Webmentionの発見情報

ルートページは、Webmention エンドポイントを `Link` ヘッダと `link` 要素で告知する SHALL。

#### Scenario: 発見情報の出力
- **WHEN** 取得側がルートページを取得する
- **THEN** システムは `rel="webmention"` の `Link` ヘッダと `link` 要素に、公開URLを起点としたエンドポイントを示す

### Requirement: 説明の正本

ルートページの説明の正本は、リポジトリ内の単一のMarkdown文書とする SHALL。表示HTMLはその文書から生成する SHALL。開発者はコードを変更せずに文書を編集できる SHALL。

#### Scenario: 正本の編集反映
- **WHEN** 開発者がMarkdownの正本を編集して実行ファイルを再ビルドする
- **THEN** ルートページの表示は編集後の内容に変わる
