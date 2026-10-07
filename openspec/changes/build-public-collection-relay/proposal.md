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
- リレーはsource URLごとの最初の検証成功スナップショットを固定し、保存済みsourceの再通知では再取得・上書き・所属移動・撤回を行わない。
- 運用者はホストの `./data` をバインドマウントした単一のリレーコンテナを起動する。
- Composeは公開経路として別のcloudflaredコンテナを起動する。ホストへのポート公開とホスト上のリバースプロキシは不要とする。
- Composeは環境変数 `CLOUDFLARED` のトークンをcloudflaredの `TUNNEL_TOKEN` に渡す。
- 運用者はCloudflare側で公開ホスト名の転送先を `http://relay:8080` に設定する。

本変更は投稿クライアント、GUI、非公開のあだ名、複数リレーの収集・マージ、原本への追従更新、所属移動、撤回、通知ごとの履歴追加を含まない。リレーは投稿の原本をホストしない。通知処理は非同期のため、保存する内容は通知到着時の原本ではなく、取得・検証した時点の内容とする。

## Capabilities

### New Capabilities

- `webmention-ingestion`: 公開受付、初回のリンク検証、スナップショットの固定、再通知の無操作。
- `collection-records`: UUID URNの分類、h-entryの選択、HTMLを含まないmf2の永続化。
- `collection-feeds`: collection別のAtomと安定したfeed・entry識別子。
- `safe-source-fetching`: 接続先・リダイレクトの検査と取得資源の上限。
- `single-container-runtime`: 単一のリレーコンテナ、cloudflaredによる公開、ホストディレクトリでの永続化、再起動後の継続処理。

### Modified Capabilities

なし。既存の仕様は存在しない。

## Impact

本変更の初稿ではGoアプリ、依存の固定、テスト、Dockerfile、運用説明を新規追加した。作業ツリーには旧仕様の実装がある。今回の改訂は追従更新・所属移動・撤回の処理を除去し、初回保存とホストディレクトリによる永続化へ揃える。旧仕様の検証結果は新仕様の完了証拠にしない。

設計候補はGo標準HTTPサーバー、 `willnorris.com/go/microformats`、 `doyensec/safeurl`、 `gorilla/feeds` のAtom専用型、SQLiteとする。部品検証はmf2とJSONとAtomの接続を確認した。部品検証はSQLite、Docker、Webmention全体を確認していない。

合意済みの範囲は公開リレー、単一のリレーコンテナ、mf2必須、h-entry、UUID URN、HTMLを持たない内部mf2モデル、source URLごとの最初の検証成功スナップショットの固定、ホストディレクトリでの永続化とする。安全な取得と有限の資源上限は維持する。

今回の配備改訂はComposeへcloudflaredを追加する。単一コンテナの条件はリレー本体に適用する。配備全体はリレーとcloudflaredの二つの常駐コンテナとする。ホストの `./data` と権限管理、バックアップは維持する。ホストへのcloudflared導入、サービス管理、リバースプロキシ、TLS終端管理は不要とする。
