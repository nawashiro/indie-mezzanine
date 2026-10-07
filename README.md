# Indiemezzanine

Indiemezzanineは、公開した記事をコレクション別のAtomフィードにまとめるアプリです。自分のサイトに記事を置き、[Webmention](https://indieweb.org/Webmention)で知らせると、読者がフィードで購読できます。

## 思想

[@moja.blue](https://tangled.org/moja.blue)さんが考案した「中2階」のIndieweb実装です。[中2階について](https://nawashiro.dev/posts/20261005-mezzanine)をご覧ください。

## 起動

### 1. 公開

デフォルトのComposeではCloudflare Tunnelを使用します。

1. Cloudflare側で公開ホスト名の転送先を `http://relay:8080` に設定してください。
2. 公開Webmention受付とAtom購読を公開してください。
   `Cloudflare > セキュリティ > セキュリティルール > カスタムルール` を開いてください。
   ホスト名・GETとPOST を選択し、WAFやブラウザ整合性チェックなどをスキップしてください。

### 2. 環境変数

[.env.example](.env.example)が設定例です。コピーして利用してください。

```sh
cp .env.example .env
```

### 3. 永続化ディレクトリ

UID/GID 65532、権限0750の保存先ディレクトリを作ってください。

```sh
sudo install -d -o 65532 -g 65532 -m 0750 ./data
```

### 4. Dockerコンテナ

〆です。

```sh
docker compose config --quiet # 設定チェック
docker compose up -d --build # 立てる
docker compose exec relay /mezzanine healthcheck # リレーのHTTP、保存先、workerチェック
```

## 投稿

[ページ例](examples/post.html)をご覧ください。

- ページはMicroformat2でマークアップしてください。Indiemezzanineは1ページに1件の[h-entry](https://microformats.org/wiki/h-entry)を読みます。
- 中2階はUUIDで区別します。ページに中2階への`rel=collection`を置いてください。`<link rel="collection" href="bbe44bfe-3a9c-410e-ad37-f1bf7402bce4">`
- `PUBLIC_URL` へのリンクを置いてください。`<a href="https://relay.example/"></a>`

リンク宛に出版ソフトやツールを使って[Webmention](https://indieweb.org/Webmention)を送信してください。

- 受け付けると`HTTP 202`を返します。
- 最初の検証に成功した内容のみ保存します。更新には対応していません。

読者は、著者が`rel=collection`で指定したUUIDをRSSリーダーなどで購読できます。

```text
https://relay.example/collections/bbe44bfe-3a9c-410e-ad37-f1bf7402bce4.atom
```

## バックアップ

`docker compose stop relay` で停止し、保存先全体をコピーしてください。コピー後は `docker compose start relay` で再開します。

コンテナの再作成には `docker compose up -d --force-recreate` を使ってください。
同じ保存先を使う場合、投稿と受付済み通知は残ります。

復元時も停止し、保存先全体をバックアップで置き換えてください。UID/GID 65532の書き込み権限を保ってから再開します。

## ライセンス

- 本体は [MIT License](LICENSE) です。
- 依存物の権利表示は[第三者通知](docs/THIRD_PARTY_NOTICES.txt)を参照してください。
- コンテナのOS資産には元のライセンス条件が適用されます。
