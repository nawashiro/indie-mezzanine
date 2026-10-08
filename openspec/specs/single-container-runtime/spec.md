# single-container-runtime Specification

## Purpose

運用者は外部のデータベースやキューサーバーを用意せず、公開リレー本体を一つのDockerコンテナとして動かす。Composeは別のcloudflaredコンテナでリレーを公開する。ホストのデータディレクトリは検証済みスナップショットと受付済み通知を保持し、コンテナの再作成後も処理を継続する。

## Requirements

### Requirement: 単一コンテナでの起動
リレーは一つのDockerコンテナでHTTP受付、投稿処理、保存、Atom配信を提供する SHALL。リレーは別コンテナのデータベース、Redis、SMTPを必要としない SHALL。運用者は公開URLとデータディレクトリを明示する SHALL。単一コンテナの条件はリレー本体に適用し、公開経路のcloudflaredコンテナを含めない SHALL。

#### Scenario: 新規起動
- **WHEN** 運用者が公開URLと書き込み可能なホストのデータディレクトリをバインドマウントして起動する
- **THEN** コンテナはデータストアを初期化する
- **AND** リレーはHTTP受付とAtom配信を開始する

#### Scenario: 設定不備
- **WHEN** 公開URLが不正、またはデータディレクトリへ書き込めない
- **THEN** リレーは原因を示して起動を失敗させる
- **AND** リレーは一時メモリ保存へ切り替えない

### Requirement: cloudflaredによる公開
Composeはリレーと別のcloudflaredサービスを起動する SHALL。cloudflaredはComposeネットワーク経由で `http://relay:8080` へ転送する SHALL。ComposeはホストへのHTTPポート公開、ホストネットワーク、Dockerソケットを使わない SHALL。配備はホスト上のcloudflaredサービス、リバースプロキシ、TLS終端を必要としない SHALL。

#### Scenario: 公開HTTPSからの接続
- **WHEN** 運用者がCloudflare側の公開ホスト名の転送先を `http://relay:8080` に設定してComposeを起動する
- **THEN** 公開HTTPSへの接続はcloudflaredからリレーへ到達する
- **AND** ComposeはホストへHTTPポートを公開しない

#### Scenario: 公開URLと公開受付
- **WHEN** 運用者がTunnelでリレーを公開する
- **THEN** 運用者は公開HTTPSルートURLを `PUBLIC_URL` に指定する
- **AND** 公開経路はWebmention受付とAtom購読に対話認証を要求しない

#### Scenario: Tunnelの停止
- **WHEN** cloudflaredが停止し、リレーが正常に動作する
- **THEN** リレーは保存済みデータを保持する
- **AND** リレーのヘルス確認はTunnelの接続状態を保証しない

### Requirement: Tunnelトークンの受け渡し
Composeは環境変数 `CLOUDFLARED` のトークンをcloudflaredの `TUNNEL_TOKEN` に渡す SHALL。Composeはトークンをrelayまたはコマンド引数へ渡さない SHALL。運用文書と検証記録はトークンの値を含まない SHALL。

#### Scenario: トークンの指定
- **WHEN** 運用者が空でない `CLOUDFLARED` を指定する
- **THEN** cloudflaredは `TUNNEL_TOKEN` からトークンを受け取る
- **AND** relayの環境変数はTunnelトークンを含まない

#### Scenario: トークンの欠落
- **WHEN** `CLOUDFLARED` が未設定または空である
- **THEN** Composeは設定エラーで起動を拒否する
- **AND** 設定エラーはトークンの値を出力しない

### Requirement: 投稿と通知の永続性
リレーは検証済みmf2、collection、処理状態、受付済み通知をホストのデータディレクトリに保存する SHALL。Composeは既定のホスト保存先 `./data` をコンテナの `/data` へバインドマウントする SHALL。Composeは `HOST_DATA_DIR` による保存先変更を維持する SHALL。ComposeはDockerの名前付きボリュームを永続化先にしない SHALL。リレーは再起動後に中断した通知処理を再開する SHALL。リレーは再処理で投稿を重複または上書きしない SHALL。

#### Scenario: コンテナの再作成
- **WHEN** 運用者が同じホストのデータディレクトリでコンテナを再作成する
- **THEN** 既存collectionと投稿は同じ識別子で配信を続ける

#### Scenario: 通知処理中の停止
- **WHEN** 受付後かつ掲載前にコンテナが停止する
- **THEN** 次回起動のリレーはその通知を再処理する
- **AND** 最終feedは投稿を一件だけ含む

#### Scenario: 掲載後かつjob完了前の停止
- **WHEN** 初回保存後かつjob完了前にコンテナが停止する
- **THEN** 次回起動のリレーは保存済みsourceの取得を省略する
- **AND** スナップショットを変更せずjobを完了させる

#### Scenario: ホスト側のバックアップと復元
- **WHEN** 運用者が停止後のホストディレクトリをバックアップし、その内容を復元して起動する
- **THEN** 保存済みスナップショットは同じ識別子と内容で配信される
- **AND** 保存済みsourceは再取得されず、未掲載の受付済み通知だけが処理される

### Requirement: 非root運用とヘルス確認
リレーコンテナは非rootユーザーで動く SHALL。リレーは稼働確認用のHTTPエンドポイントを提供する SHALL。リレーは投稿を正常処理しない状態で正常な稼働状態を返さない SHALL。

#### Scenario: 正常稼働
- **WHEN** HTTP処理、保存先、投稿処理の実行系が稼働する
- **THEN** ヘルス確認はHTTP 200を返す

#### Scenario: 保存先の利用不能
- **WHEN** 保存先が利用不能になる
- **THEN** ヘルス確認は非成功ステータスを返す
- **AND** リレーは保存していない通知へHTTP 202を返さない
