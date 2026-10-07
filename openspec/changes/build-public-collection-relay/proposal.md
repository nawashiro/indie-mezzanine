# Proposal

## Why

投稿者は自分の公開ページを、共通の識別子で小さな共有空間に束ねる。mezzanineは既存のWebmentionとmicroformats2を使い、この集合をAtomとして配信する公開リレーを提供する。

## What Changes

- リレーは認証や手動承認なしでWebmentionを受け付ける。
- リレーはsourceを取得し、targetへのリンクを検証する。
- リレーはmicroformats2（mf2）の `h-entry` を投稿として扱う。
- 初版のリレーは1ページにつき1つのh-entryと1つのcollectionだけを受け付ける。
- リレーは `rel="collection"` のUUID URNだけを所属先として受け付ける。
- リレーはmf2の入れ子構造を保持し、HTML表現を保存前に除去する。
- リレーはプレーンテキストのコンテンツからcollection別のAtomを生成する。
- リレーは更新・削除の再通知を反映し、同じ通知による重複を防ぐ。
- 運用者はデータ用ボリュームを付けた単一Dockerコンテナを起動する。

本変更は投稿クライアント、GUI、非公開のあだ名、複数リレーの収集・マージを含まない。リレーは投稿の原本をホストしない。

## Capabilities

### New Capabilities

- `webmention-ingestion`: 公開受付、リンク検証、再通知、投稿の更新・撤回。
- `collection-records`: UUID URNの分類、h-entryの選択、HTMLを含まないmf2の永続化。
- `collection-feeds`: collection別のAtomと安定したfeed・entry識別子。
- `safe-source-fetching`: 接続先・リダイレクトの検査と取得資源の上限。
- `single-container-runtime`: 単一コンテナ、永続データ、再起動後の継続処理。

### Modified Capabilities

なし。既存の仕様は存在しない。

## Impact

現状のリポジトリはREADMEとOpenSpec初期化ファイルだけを含む。実装はGoアプリ、依存の固定、テスト、Dockerfile、運用説明を新規追加する。

設計候補はGo標準HTTPサーバー、 `willnorris.com/go/microformats`、 `doyensec/safeurl`、 `gorilla/feeds` のAtom専用型、SQLiteとする。部品検証はmf2とJSONとAtomの接続を確認した。部品検証はSQLite、Docker、Webmention全体を確認していない。

合意済みの範囲は公開リレー、単一コンテナ、mf2必須、h-entry、UUID URN、HTMLを持たない内部mf2モデルとする。エンドポイント、投稿選択、削除判定、資源上限の詳細は本提案のレビュー対象とする。設計は未合意の判断を明示する。
