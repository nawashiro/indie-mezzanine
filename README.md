# mezzanine

mezzanineは「中二階」の公開リレーを実装する。
構想の出典: https://nawashiro.dev/posts/20261005-mezzanine

投稿者は自分の公開ページをUUID URNで束ねる。
リレーはWebmention通知を受け付ける。
リレーは原本のリンクと一つのh-entryを確認する。
リレーはHTMLを除去したmicroformats2をSQLiteへ保存する。
購読者はcollection別のAtomを読む。

初版は一つのページと一つのcollectionだけを扱う。
投稿クライアント、GUI、非公開のあだ名、複数リレーの収集とマージは対象外とする。
リレーはsource URLごとの最初の検証成功スナップショットを固定する。
リレーは再通知でも原本の更新・所属移動・削除を反映しない。
スナップショットは通知後の取得・検証時点の内容とする。

## Goで起動

開発者は Go 1.27.1 を用意する。
開発者は次のコマンドでビルドして起動する。

```sh
mkdir -p bin
CGO_ENABLED=0 go build -trimpath -o bin/mezzanine ./cmd/mezzanine
PUBLIC_URL=https://relay.example/ DATA_DIR=./data LISTEN_ADDR=:8080 ./bin/mezzanine
```

`PUBLIC_URL` は実際の公開ルートURLに置き換える。
開発者は別のターミナルで稼働を確認する。

```sh
./bin/mezzanine healthcheck
curl --fail http://127.0.0.1:8080/healthz
```

## Dockerで起動

```sh
cp .env.example .env
# 運用者は .env の PUBLIC_URL を変更する。
# HOST_DATA_DIR を変更する場合、運用者は以下も同じ保存先へ置き換える。
sudo install -d -o 65532 -g 65532 -m 0750 ./data
docker compose config
docker compose up -d --build
```

Dockerfileは非rootの単一コンテナを作る。
Composeはホストの `./data` を `/data` へバインドマウントする。
運用者は `HOST_DATA_DIR` でホスト側の保存先を変更する。
Composeは存在しないホストディレクトリを自動作成しない。
ComposeはHTTPを127.0.0.1:8080へ公開する。
運用者は既存のプロキシで公開HTTPSを終端する。
Docker実行検証の未完了事項は検証記録を参照する。

## 文書

- [投稿・通知・購読](docs/PUBLISHING.md)
- [設定・再作成・バックアップ・復元](docs/OPERATIONS.md)
- [依存とライセンス](docs/DEPENDENCIES.md)
- [実行した検証と未完了事項](docs/VERIFICATION.md)
- [投稿ページ例](examples/post.html)

## 検証

テストは独立した固定fixtureを使う。
テスト専用Fetcherはローカルsourceを読む。
本番の取得設定は内部ネットワークを許可しない。
文書のcurl検証にはcurlとPOSIX shellを使う。

```sh
go mod verify
go vet ./...
go test -race ./...
CGO_ENABLED=0 go build ./...
openspec validate build-public-collection-relay --strict
```

## 仕様管理

OpenSpecはspec-driven方式を使う。
`openspec/changes/build-public-collection-relay/` は初版の仕様差分、設計、実装タスクを保持する。
Docker実行の未検証タスクは未完了のまま残す。
アーカイブは全ての完了条件を検証した後に扱う。

## ライセンス

mezzanine本体は [MIT License](LICENSE) で公開する。
第三者の依存物は元のライセンス条件を保持する。
配布時の区別は [ライセンス方針](docs/LICENSING.md) を参照する。
コンテナのDebian資産は本体のMITとは別条件とする。
