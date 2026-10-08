# Proposal

## Why

READMEはシステム名を `indie-mezzanine` と記す。実装、配備設定、設計書は旧名を使い、読者と運用者に異なる名前を示す。

## What Changes

- 公開ページ、Atomのフィード著者名、HTTP User-Agent、起動ログの製品名を `indie-mezzanine` に統一する。
- **BREAKING** Goモジュール、コマンドディレクトリ、実行ファイルを `indie-mezzanine` に改名する。
- **BREAKING** Composeプロジェクト名を `indie-mezzanine`、ローカルイメージ名を `indie-mezzanine:local` に変更する。
- **BREAKING** 保存ファイル名を `indie-mezzanine.db` に変更する。実装は旧名の探索、自動移行、別名対応を提供しない。
- テスト専用識別子を新名の表記へ揃える。運用例とOpenSpecの現行製品名も揃える。
- 元の概念名「中2階／中二階」、役割名 `relay`、外部URL、過去の資料パス、既存change識別子を保持する。
- collectionのUUID、AtomのIDとタイトル、投稿者情報、HTTP経路、設定変数の契約を維持する。

## Capabilities

### New Capabilities

- `system-identity`: 公開出力、実行入口、配備、保存先におけるシステム名と命名規則を定義する。

### Modified Capabilities

なし。`openspec list --specs` は既存の正本仕様を返さない。初期changeの仕様は未アーカイブであり、本changeは独立した命名契約を追加する。

## Impact

- `internal/relay/{server,feed,fetch,store}.go`、`cmd/mezzanine/main.go`、`go.mod` が対象となる。
- `Dockerfile`、`compose.yml`、`README.md`、`openspec/config.yaml` が対象となる。
- バックアップ復元テスト、取得テスト、Composeテストと新しい命名回帰テストが対象となる。
- 初期changeのproposalとdesignにある現行製品名を更新する。過去の資料参照は保持する。
- 依存ライブラリ、DBスキーマ、Cloudflare設定、稼働中の配備、ローカルのリポジトリ格納ディレクトリは変更しない。
- 旧DBと旧Composeプロジェクトの継続利用は保証しない。実装者は旧資産を自動削除しない。
