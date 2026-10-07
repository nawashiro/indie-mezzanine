# mezzanine

mezzanineは、公開ページをWebmentionで受け付け、コレクション別のAtomフィードへまとめる小さなリレー。
source URLごとの最初の検証成功スナップショットを保存する。原本の更新・所属変更・削除には追従しない。

構想: https://nawashiro.dev/posts/20261005-mezzanine

## 起動

Docker EngineとDocker Composeを使う。

```sh
cp .env.example .env
# .envのPUBLIC_URLを実際の公開ルートURLへ変更する。
sudo install -d -o 65532 -g 65532 -m 0750 ./data
docker compose up -d --build
docker compose exec relay /mezzanine healthcheck
```

ComposeはHTTPを `127.0.0.1:8080` に公開する。公開HTTPSは既存のリバースプロキシで終端する。
`PUBLIC_URL` はサブパスを含まないルートURLにする。
保存先はホストの `./data`。別の保存先を使う場合、`.env` の `HOST_DATA_DIR` と作成するディレクトリを揃える。
SELinuxが有効なホストでは、保存先にコンテナ用ラベルも設定する。

## 投稿と購読

投稿者は [ページ例](examples/post.html) のUUIDとリレーへのリンクを変更し、HTMLを公開する。
一つのページに一つの `h-entry` と一つのコレクションを置く。

```sh
curl --fail-with-body -i \
  --data-urlencode 'source=https://author.example/post' \
  --data-urlencode 'target=https://relay.example/' \
  https://relay.example/webmention
```

202は通知の受付を示す。リレーは原本を取得・検証した後に掲載する。
保存済みsourceの再通知は200になる。リレーはその原本を再取得しない。
購読者はコレクションのUUIDを使う。

```text
https://relay.example/collections/550e8400-e29b-41d4-a716-446655440000.atom
```

## バックアップ

運用者は `docker compose stop relay` で停止し、保存先のディレクトリ全体をコピーする。
コピー後は `docker compose start relay` で再開する。
復元時も停止し、ディレクトリ全体をバックアップで置き換える。運用者はUID/GID 65532の書き込み権限を保つ。

## 開発

Go 1.27.1を使う。仕様とタスクは `openspec/`、開発用スキルは `.hermes/skills/` に置く。

```sh
go test -race ./...
CGO_ENABLED=0 go build -o bin/mezzanine ./cmd/mezzanine
PUBLIC_URL=https://relay.example/ DATA_DIR=./data LISTEN_ADDR=:8080 ./bin/mezzanine
```

## ライセンス

本体は [MIT License](LICENSE)。依存物の権利表示は [第三者通知](docs/THIRD_PARTY_NOTICES.txt) に保持する。
コンテナのOS資産には元のライセンス条件を適用する。イメージ全体をMITだけとは扱わない。
