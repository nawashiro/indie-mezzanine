# mezzanine

mezzanineは、公開した記事をコレクション別のAtomフィードにまとめるアプリです。自分のサイトに記事を置き、Webmention（ページ間のリンク通知）で知らせると、読者がフィードで購読できます。

## 起動

Docker EngineとDocker Compose、公開URL、HTTPSを終端するリバースプロキシを用意してください。

```sh
cp .env.example .env
```

`.env` の `PUBLIC_URL` を実際の公開ルートURL（例: `https://relay.example/`）に変更してください。サブパス・認証情報・クエリ・フラグメントは含めません。

```sh
sudo install -d -o 65532 -g 65532 -m 0750 ./data
docker compose up -d --build
docker compose exec relay /mezzanine healthcheck
```

HTTPの接続先は既定で `127.0.0.1:8080` です。リバースプロキシからここへ転送してください。

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

復元時も停止し、保存先全体をバックアップで置き換えてください。UID/GID 65532の書き込み権限を保ってから再開します。

## ライセンス

本体は [MIT License](LICENSE) です。依存物の権利表示は[第三者通知](docs/THIRD_PARTY_NOTICES.txt)を参照してください。
コンテナのOS資産には元のライセンス条件が適用されます。イメージ全体がMITライセンスではありません。
