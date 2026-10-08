# Spec Delta

## Purpose

本仕様は、読者、運用者、取得先サーバーへ示すシステム名を統一する。公開出力、実行入口、配備資産、保存ファイルにおける命名契約を定義し、投稿やcollectionの識別情報と区別する。

## ADDED Requirements

### Requirement: 公開出力のシステム名

システムは製品名として `indie-mezzanine` を使用する（SHALL）。ルートページのtitleと見出し、Atomのフィード著者名は `indie-mezzanine` とする（SHALL）。取得時のUser-Agentは `indie-mezzanine Webmention receiver` とする（SHALL）。起動ログは `indie-mezzanine listening on` で開始する（SHALL）。

#### Scenario: ルートページの表示
- **WHEN** 読者がルートページを取得する
- **THEN** システムはtitleと主見出しに `indie-mezzanine` を表示する

#### Scenario: Atomの著者表示
- **WHEN** 読者が既知のcollectionのAtomを取得する
- **THEN** システムはフィード著者名に `indie-mezzanine` を出力する
- **AND** システムはentryの投稿者情報を維持する

#### Scenario: 原本への取得要求
- **WHEN** システムが検証対象ページを取得する
- **THEN** システムはUser-Agentに `indie-mezzanine Webmention receiver` を送る

#### Scenario: 起動の記録
- **WHEN** システムがHTTPサーバーを起動する
- **THEN** システムは `indie-mezzanine listening on` と待受アドレスをログへ記録する

### Requirement: 実行と配備の名前

配布物は実行ファイル名として `indie-mezzanine` を使用する（SHALL）。コンテナは `/indie-mezzanine` を起動入口とヘルスチェックに使用する（SHALL）。Composeはプロジェクト名 `indie-mezzanine` とイメージ名 `indie-mezzanine:local` を使用する（SHALL）。運用文書は同じ実行入口を示す（SHALL）。

#### Scenario: Compose設定の解決
- **WHEN** 運用者が必須環境変数を設定してCompose設定を解決する
- **THEN** Composeはプロジェクト名 `indie-mezzanine` とリレーイメージ名 `indie-mezzanine:local` を返す
- **AND** リレーのヘルスチェックは `/indie-mezzanine healthcheck` を実行する

#### Scenario: コンテナ実行
- **WHEN** 運用者が新しいリレーイメージを起動する
- **THEN** コンテナは `/indie-mezzanine` を実行する
- **AND** 運用者は同じ実行ファイルの `healthcheck` で稼働状態を確認する

### Requirement: 保存ファイルの名前

システムは `DATA_DIR` の `indie-mezzanine.db` をSQLite保存ファイルとして使用する（SHALL）。システムは旧名の保存ファイルを探索、移行、改名、削除しない（MUST NOT）。システムは旧名へのフォールバックを提供しない（MUST NOT）。

#### Scenario: 新規保存先の起動
- **WHEN** システムが空の保存先で起動する
- **THEN** システムは `indie-mezzanine.db` を作成する
- **AND** システムは旧名のDBを作成しない

#### Scenario: 旧名ファイルの存在
- **WHEN** 保存先に旧名のDBだけが存在する
- **THEN** システムは新名のDBを使用する
- **AND** システムは旧DBの内容を読み込まず、旧ファイルを変更しない

#### Scenario: 新名によるバックアップ復元
- **WHEN** 運用者が停止後の新名DBを復元してシステムを起動する
- **THEN** システムは保存済み投稿と受付済み通知を読み戻す

### Requirement: 改名とプロトコル識別情報の分離

システムは製品名の統一によってcollectionのUUID、AtomのIDとタイトル、投稿者情報、HTTP経路を変更しない（MUST NOT）。Composeは役割名 `relay` と `cloudflared` を保持する（SHALL）。運用文書は概念名「中2階／中二階」と既存の外部参照先を保持する（SHALL）。

#### Scenario: collectionと投稿の配信
- **WHEN** システムが既存形式の投稿を新名DBへ保存して配信する
- **THEN** システムはcollectionの正規URNをフィードのIDとタイトルに使用する
- **AND** システムはsource URLをentryのIDとして使用する
- **AND** システムは `/collections/{uuid}.atom` で配信する

#### Scenario: Tunnelの接続先
- **WHEN** 運用者が新名のCompose構成を使用する
- **THEN** リレーのサービス名は `relay` のままとなる
- **AND** Tunnelの転送先は `http://relay:8080` のままとなる
