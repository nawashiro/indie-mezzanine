# indie-mezzanine

indie-mezzanineは、公開した記事をコレクション別のAtomフィードにまとめるアプリです。自分のサイトに記事を置き、[Webmention](https://indieweb.org/Webmention)で知らせると、読者がフィードで購読できます。

## 思想

[@moja.blue](https://tangled.org/moja.blue)さんが考案した「中二階」のIndieweb実装です。解説は[中二階について - Nawashiro](https://nawashiro.dev/posts/20261005-mezzanine)をご覧ください。

## 起動

運用者はDocker Engine、Docker Compose、Cloudflareの公開ホスト名、リモート管理Tunnelを用意してください。
Composeはリレーとcloudflaredを起動します。Cloudflareは公開HTTPSを終端します。
ホストへのcloudflared導入、リバースプロキシ、TLS証明書管理は不要です。

Cloudflare側で公開ホスト名の転送先を `http://relay:8080` に設定してください。
`localhost` はcloudflared自身を指すため、転送先に指定しないでください。
公開Webmention受付とAtom購読には、Cloudflare Accessのログインや対話型チャレンジを要求しないでください。
ブラウザー整合性チェックがWebmention送信者やフィードリーダーを拒否する場合、公開ホスト名への適用を除外してください。

```sh
cp .env.example .env
```

`.env` の `PUBLIC_URL` を同じ公開ホスト名のHTTPSルートURL（例: `https://relay.example/`）に変更してください。
サブパス・認証情報・クエリ・フラグメントは含めません。

運用者はTunnelトークンを環境変数 `CLOUDFLARED` に指定してください。
Composeはシェルの環境変数を優先します。運用者は未追跡の `.env` でも指定できます。
未設定または空の値は、Composeの起動エラーになります。
Composeは値をcloudflaredの `TUNNEL_TOKEN` へ渡します。リレーはトークンを受け取りません。

トークンをコマンド引数、ログ、バージョン管理へ記録しないでください。
`docker compose config` の出力はトークンを含むため、表示・保存しないでください。
設定検証には `docker compose config --quiet` を使ってください。
Docker管理権限を持つ利用者はコンテナの環境変数を参照できます。

```sh
sudo install -d -o 65532 -g 65532 -m 0750 ./data
docker compose config --quiet
docker compose up -d --build
docker compose exec relay /mezzanine healthcheck
```

ComposeはホストへHTTPポートを公開しません。
cloudflaredはComposeネットワーク経由で `relay:8080` へ接続します。
運用者は公開URLでルートページとWebmention endpointの広告を確認してください。

`docker compose exec relay /mezzanine healthcheck` はリレーのHTTP、保存先、workerを確認します。
この確認はTunnelの接続状態や公開到達性を保証しません。
公開URLへ接続できない場合、Cloudflare側の転送先とTunnel接続状態を確認してください。
cloudflaredのイメージ更新は、Composeの固定バージョンとdigestの更新で管理します。

保存先はホストの `./data` です。変更する場合は、`.env` の `HOST_DATA_DIR` と作成するディレクトリを揃えてください。SELinuxが有効なホストでは、保存先にコンテナ用ラベルも設定してください。

## 投稿と購読

[ページ例](examples/post.html)のコレクションUUIDとリレーへのリンクを変更し、記事のHTMLを公開してください。一つのページに一つの `h-entry` と一つのコレクションを置きます。

`PUBLIC_URL` へのリンクを含め、Webmentionで通知してください。

```html
<a href="https://relay.example/"></a>
```

HTTP 202は受付を示し、掲載の保証ではありません。原本を取得・検証してから掲載します。保存済みの記事の再通知にはHTTP 200を返し、原本を再取得しません。

読者は以下のURLをフィードリーダーに登録します。ホスト名とUUIDは、投稿先と記事に指定した値に置き換えてください。

```text
https://relay.example/collections/550e8400-e29b-41d4-a716-446655440000.atom
```

最初の検証に成功した内容のみ保存します。更新には対応していません。

## バックアップ

`docker compose stop relay` で停止し、保存先全体をコピーしてください。コピー後は `docker compose start relay` で再開します。

コンテナの再作成には `docker compose up -d --force-recreate` を使ってください。
同じ保存先を使う場合、投稿と受付済み通知は残ります。

復元時も停止し、保存先全体をバックアップで置き換えてください。UID/GID 65532の書き込み権限を保ってから再開します。

## ライセンス

本体は [MIT License](LICENSE) です。依存物の権利表示は[第三者通知](docs/THIRD_PARTY_NOTICES.txt)を参照してください。
コンテナのOS資産には元のライセンス条件が適用されます。イメージ全体がMITライセンスではありません。
