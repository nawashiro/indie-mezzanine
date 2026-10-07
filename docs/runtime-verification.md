# コンテナ配備の検証記録

## 実行環境

実装者はDocker Engine 29.7.2とDocker Compose 5.5.1で検証した。
ホストにはGoがない。実装者はDockerfileのbuildステージでGoの検査を実行した。

Dockerの通常ビルドは、ホスト指定のIPv6 DNSサーバーへの接続で失敗した。
実装者は利用者の許可で検証ビルドだけに `--network=host` を指定した。
配備用Composeはホストネットワークを使わない。

その後、利用者はDockerデーモンのDNSを `8.8.8.8` と `8.8.4.4` に設定した。
利用者は既存のランタイム設定を維持してDockerを再起動した。
実装者は `docker compose build --no-cache relay` の成功を確認した。
この再検証はホストネットワークを使わない。依存取得とモジュール検証も成功した。
再起動後のリレーのhealth、公開ルート、保存済みAtomは正常だった。
現在のビルドは `--network=host` を必要としない。

## 成功した検証

- `docker build --network=host -t mezzanine:local .`
- `docker build --network=host --target build -t mezzanine:build-check .`
- `python3 tests/compose_test.py`：4件成功。
- buildステージ内の `go vet ./...` と `go test -race ./...`。
- `openspec validate build-public-collection-relay --strict`。

Goの検査はネットワークなしのコンテナで実行した。
実装者は `docker cp` で `examples/` を検査コンテナの `/src/examples/` へ追加した。

Composeの回帰検証はダミートークンだけを使う。
検証は `.env` を読み込まない。検証は展開した設定を表示しない。
検証はポート非公開、共通ネットワーク、バインドマウント、イメージ固定、トークンの受け渡しを確認する。
未設定・空のトークンは設定エラーになる。

実装者は一時ディレクトリでComposeのリレーを起動した。
検証用保存先はUID/GID 65532、モード0750とした。
リレーは非rootで保存先へ書き込んだ。内部のサービス名 `relay:8080` へのヘルス確認も成功した。
リレーはホストへポートを公開しない。読み取り専用の保存先ではリレーの起動が失敗した。
cloudflaredはComposeの非root・読み取り専用設定で `--version` を実行した。

実装者は停止後のDBへ固定fixtureの投稿と通知を設定した。
fixtureは保存済み投稿、掲載後のrunning通知、将来実行予定のpending通知を含む。
再作成後、保存済み投稿の全属性は不変だった。running通知はdoneへ進んだ。
保存済みsourceは取得禁止の内部IPを指定するため、doneへの遷移は取得省略を示す。

実装者はリレーを停止し、保存先全体をコピーした。
実装者はコピーを復元し、UID/GID 65532を設定してリレーを再作成した。
復元後、投稿の全属性とpending通知は残った。ヘルス確認も成功した。
実装者は実保存先 `./data` と実Tunnelトークンを検証に使わなかった。

## 実Tunnelの起動と公開経路

利用者は実保存先の準備と公開検証を許可した。
実装者はホストの `./data` をUID/GID 65532、モード0750で作成した。
SELinux用の共有コンテナラベルを設定した。
実装者は `docker compose up -d --no-build` で両サービスを起動した。
リレーのhealthは正常だった。cloudflaredは4接続を登録した。
検証はトークンを表示・記録しなかった。

公開URLは `https://indiemezzanine.nawashiro.dev/` とする。
公開URLの応答はUser-Agentで異なった。

| User-Agent | ルート | 空フォームのWebmention | 未知collectionのAtom |
| --- | --- | --- | --- |
| `Python-urllib/3.14` | 403 | 403 | 403 |
| `Mozilla/5.0` | 200 | 400 | 404 |

200のルートはHTTP LinkヘッダーとHTMLでWebmention endpointを広告した。
403の応答は `Server: cloudflare` を返した。応答はchallengeの表示を含まなかった。
初回の検証時点では、実装者はCloudflare側の拒否ルールを特定しなかった。
User-Agentの変更は自動クライアントの公開利用を保証しない。

実装者は実保存先へ検証投稿を追加しなかった。
実装者は両サービスを起動した状態で検証を中断した。
利用者はブラウザー整合性チェックによる拒否を確認した。
利用者はCloudflare側の設定を変更した。
変更後、両User-Agentのルートは200を返した。HTTP LinkヘッダーとHTMLの広告も確認した。
両User-Agentの空フォームPOSTはアプリ由来の400を返した。
両User-Agentの未知collectionは404を返した。
自動クライアントの403は解消した。

## 公開テスト記事の通知

利用者は固定テスト記事 `https://nawashiro.dev/posts/20261007-test-of-mezzanine` を指定した。
実装者はリレーへのリンク、単一h-entry、collection宣言を確認した。
公開Webmentionは202を返した。Atomへの初回掲載は確認できなかった。

実装者は同じComposeネットワークで本番のSafeFetcherを実行した。
取得は `malformed HTTP response` で失敗した。応答先頭はHTTP/2のバイナリフレームだった。
Go標準HTTPクライアントとリレーと同じUser-Agentによる取得は200だった。
利用者は取得層の修正を許可した。
実装者は `TestSafeFetcherTLSProtocols` の固定TLSサーバーで同じエラーを再現した。
safeurlはTransportの `DialContext` を差し替える。
実装者は `ForceAttemptHTTP2: true` でALPNの交渉と受信処理を一致させた。
修正後、HTTP/1.1とHTTP/2の固定fixture試験は成功した。
`go vet ./...` と `go test -race ./...` も成功した。
本番のIP・ポート・redirect制約は維持した。内部IPの許可はTLS試験の設定だけに限定した。

実装者は修正したイメージでリレーを再作成した。
公開Webmentionの再送は202を返した。公開Atomは200と `application/atom+xml` を返した。
collectionは `bbe44bfe-3a9c-410e-ad37-f1bf7402bce4` とする。
feedはテスト記事のsource URLをidとするentryを一件だけ含んだ。
entryのタイトルは「Indiemezzanineのテスト」、contentはtype=textだった。
掲載後の再通知は200を返した。再通知前後のAtom全体のSHA-256は一致した。

## コンテナの中断復旧

実装者は停止後の実DBを一時保存先へコピーした。
実装者はコピーだけから投稿を除去し、テスト記事のrunning通知を設定した。
実装者は本番イメージの別コンテナを同じネットワークで起動した。
通知は復旧後に初回掲載を完了した。投稿は一件だけだった。

実装者は同じコピーへ掲載後のrunning通知を設定した。
実装者はネットワークなしでコンテナを再作成した。
通知はdoneへ進んだ。投稿の全属性は不変だった。
ネットワークなしの成功は、保存済みsourceの取得省略を示す。
実装者は実DBへ復旧試験用の変更を加えなかった。

## Tunnelの停止と回復

実装者はcloudflaredを停止した。リレーのhealthは正常だった。
公開ルートは502を返した。
実装者はcloudflaredを再起動した。公開Atomは200へ回復した。
停止前後のAtom全体のSHA-256は一致した。
実行中のリレーはホストへポートを公開しなかった。

## 相互運用の範囲

公開試験はPythonのHTTPクライアント、利用者指定の外部HTML、実Tunnelを使った。
試験は公開広告、フォーム通知、外部原本の検証、AtomのXML解析、再通知を確認した。
固定fixtureのGo試験はリンク検証、入力拒否、SSRF制約を確認した。
試験は全Webmention実装との互換性を保証しない。
本アプリは原本の更新・削除へ追従しない。再通知は初回スナップショットを変更しない。

実装者は検証後も両サービスを起動した状態に保った。
実保存先はテスト記事のスナップショットを保持する。
