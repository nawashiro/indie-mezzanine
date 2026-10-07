# mezzanine

「中二階」の構想を検討・実装するためのリポジトリです。`mezzanine` は仮のプロジェクト名です。

構想の出典: https://nawashiro.dev/posts/20261005-mezzanine

## 目的

ランダムな識別子を使った小さな共有空間で、参加者が共通の話題をゆるく共有できるようにします。

## 検討の出発点

以下は記事から抽出した構想であり、確定した実装仕様ではありません。

- 共有空間をランダムな記号列で識別する。
- 利用者ごとに非公開のあだ名を付け、画面ではそのあだ名を表示する。
- 新しいプロトコルを作らず、Webmentionをリレーサーバーへ送る案を検討する。
- 複数リレーへの送信と、共通のAtom `id` を利用した収集・マージの案を検討する。

## 仕様管理

OpenSpecの `spec-driven` ワークフローを使います。文書の本文は日本語です。

```sh
openspec list --json
openspec new change <change-name>
openspec status --change <change-name>
```

- `openspec/config.yaml`: プロジェクトの背景と文書の方針
- `openspec/specs/`: 合意された仕様
- `openspec/changes/`: 提案、仕様差分、設計、実装タスク
- `.hermes/skills/`: OpenSpec CLIが生成したHermes用スキル

初期化のみ完了しています。アプリケーションの実装、技術スタックの選定、最初の変更提案はまだ行っていません。
