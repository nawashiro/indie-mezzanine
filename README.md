# indie-mezzanine

indie-mezzanineは、個人サイトのページを、特にテーマの決まっていないフィードに追加できる仕組みです。著者のオプトイン式、配信はAtomとJF2 Feedです。

## 思想

[@moja.blue](https://tangled.org/moja.blue)さんが考案した「中2階」のIndieweb実装です。[私の解釈はこうです。](https://nawashiro.dev/posts/20261005-mezzanine)

- フィードに共通の話題がなんとなくある。明示はされない。空気は読んでもいいし、無視してもいい。
- 小さな空間だ。SNSのような大海に放り出されない。
- 誰でも始められる。識別子はただのランダムな文字列だ。

## 投稿

[利用者向け解説](internal/relay/landing.md)を読んでください。`/`はサーバーのルートURLを指しています。例 `/webmention` → `https://example.com/webmention`

## 起動

### 1. 公開

デフォルトのComposeではCloudflare Tunnelを使用します。

1. Cloudflare側で公開ホスト名の転送先を `http://relay:8080` に設定してください。
2. 公開Webmention受付とAtom・JF2購読を公開してください。
   - `Cloudflare > セキュリティ > セキュリティルール > カスタムルール` を開いてください。
   - ホスト名・GETとPOST を選択し、WAFやブラウザ整合性チェックなどをスキップしてください。

### 2. 環境変数

[.env.example](.env.example)が設定例です。コピーして利用してください。

`PUBLIC_URL` はHTTP(S)の公開ルートURLです。リレーは末尾 `/` の有無を同一視します。認証情報、query、空でないfragment、ルート以外のパスは指定しないでください。

```sh
cp .env.example .env
```

### 3. 永続化ディレクトリ

UID/GID`65532`、権限`0750`の保存先ディレクトリを作ってください。

```sh
sudo install -d -o 65532 -g 65532 -m 0750 ./data
```

### 4. Dockerコンテナ

〆です。

```sh
docker compose config --quiet # 設定チェック
docker compose up -d --build # 立てる
docker compose exec relay /indie-mezzanine healthcheck # リレーのHTTP、保存先、workerチェック
```

## バックアップ

ルートURL比較の更新では、既存job、投稿、source識別子、SQLiteスキーマの書換えは不要です。更新前にリレーを停止し、保存先全体をバックアップしてください。同じ設定と保存先で新しいイメージを起動してください。

過去の検証失敗は自動再処理しません。未保存の記事を修正し、再通知してください。旧イメージへ戻す場合、データベースの逆移行は不要です。ただし末尾 `/` に依存する旧判定が再発します。受付と未処理通知を確認してください。

- `docker compose stop relay` で停止し、保存先全体をコピーしてください。コピー後は `docker compose start relay` で再開します。
- コンテナの再作成には `docker compose up -d --force-recreate` を使ってください。同じ保存先を使う場合、投稿と受付済み通知は残ります。

## ライセンス

- 本体は [MIT License](LICENSE) です。
- 依存物の権利表示は[第三者通知](docs/THIRD_PARTY_NOTICES.txt)を参照してください。
- コンテナのOS資産には元のライセンス条件が適用されます。
