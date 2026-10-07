# 検証記録

## 対象と実行環境

本記録は初回スナップショット固定への改訂後を対象とする。
検証者はLinux amd64とGo 1.27.1で直接実装した。
旧仕様の更新・移動・撤回試験は不変性の試験へ置き換えた。
旧仕様の検証記録は改訂前のGit履歴に残る。
回帰テストは実記事と独立した固定fixtureを使う。

## 成功した検査

```sh
go mod verify
go vet ./...
go test -race -count=3 ./...
CGO_ENABLED=0 go build -trimpath -o /opt/data/cache/mezzanine ./cmd/mezzanine
go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...
openspec validate build-public-collection-relay --strict
git diff --check
```

依存検査は `all modules verified` を返した。
race検査は全パッケージで成功した。
脆弱性検査は `No vulnerabilities found.` を返した。
OpenSpecの厳格検証は成功した。

検証者はCompose v2.39.4の設定検査とhadolint v2.12.0を実行した。
両検査の終了コードは0だった。
Composeの既定保存先はホストの `./data` だった。
`HOST_DATA_DIR` の変更はバインド先へ反映された。
Composeは名前付きボリュームを含まなかった。
バインド設定の `create_host_path` はfalseだった。
Dockerfileは匿名VOLUMEを含まない。
これらの静的検査はコンテナ実行の代わりにならない。

## 実バイナリの確認

検証者はUID 1000で非CGOバイナリを起動した。
検証者は一時保存先とループバックの一時ポートを使った。
任意ポートのhealthcheckには `HEALTH_URL` を指定した。

- ルートとhealthのHTTP: 200。
- Webmention広告: 設定PUBLIC_URL由来のendpoint。
- healthcheckコマンド: 終了コード0。
- 未保存sourceの通知受付: HTTP 202。
- SIGTERM後の停止: 終了コード0。
- 停止後のSQLite: jobを一件保持、スキーマ版2。
- 書き込み不能な保存先: 起動失敗、終了コード1。

この確認はコンテナUID 65532の検証ではない。
実バイナリの通知先は予約済みの `.invalid` ドメインを使った。
この通知試験は受付と永続化だけを確認し、掲載成功を主張しない。

## タスク別の証拠

| タスク | 状態 | 主な検査 |
| --- | --- | --- |
| 1.1 | 完了 | 非CGOビルド、UID 1000の実バイナリ起動 |
| 1.2 | 完了 | go mod verify、依存固定とライセンス一覧 |
| 1.3 | 完了 | DB保存・再接続、非CGOビルド |
| 2.1 | 完了 | TestSafeFetcherRejects、TestConfig |
| 2.2 | 完了 | TestRedirectChecks、環境proxy無効化 |
| 2.3 | 完了 | TestBodyAndTimeLimitsのgzip・chunked・遅延 |
| 2.4 | 完了 | TestFetchDNSChangeIsolated、race検査 |
| 3.1 | 完了 | TestLinkAttributes、TestParseConstraints |
| 3.2 | 完了 | TestCollection、単一h-entryと単一所属のfixture |
| 3.3 | 完了 | TestMF2StructureAndStableJSON、入れ子HTML除去とJSON往復 |
| 3.4 | 完了 | TestExampleAndDocumentCommandsの投稿例解析 |
| 4.1 | 完了 | TestStoreSchemaAndUnavailable、source単独のDB一意制約 |
| 4.2 | 完了 | TestStorePersistenceAndTransactions、TestConcurrentFirstSnapshot |
| 4.3 | 完了 | TestQueue、TestWorkerRetriesFinite、TestPermanentInitialFailure、掲載後中断復旧 |
| 4.4 | 完了 | 保存境界の説明、DB再接続、バックアップ復元と掲載前後の復旧 |
| 5.1 | 完了 | TestHTTPAdmission。未保存202、保存済み200、満杯時もjob追加ゼロ |
| 5.2 | 完了 | TestProcessOutcomes、保存後の再通知は外部取得ゼロ |
| 5.3 | 完了 | TestConfig、TestHealthAndShutdown、実バイナリのSIGTERM終了 |
| 5.4 | 完了 | 文書のcurlを実行。未保存202、掲載後の再通知200 |
| 6.1 | 完了 | TestAtom、TestDateFallback、XML解析、textとメタデータ |
| 6.2 | 完了 | TestFeedOrderingLimitAndTimes、E2EのAtom不変性と未知collectionの404 |
| 6.3 | 完了 | 購読文書、文書の購読URL取得、固定updatedのXML検査 |
| 7.1 | 未完了 | ComposeとDockerfileの静的検査成功。コンテナ実行とUID 65532は未確認 |
| 7.2 | 未完了 | 運用文書とGoの復元試験。Docker運用コマンドは未実行 |
| 8.1 | 完了 | TestE2EAndRestart。初回失敗後の成功、編集・移動・削除後の不変性 |
| 8.2 | 未完了 | GoのStore/App復旧成功。コンテナ停止・再作成は未実行 |
| 8.3 | 完了 | go vet、race 3回、OpenSpec strict、git diff --check |
| 8.4 | 未完了 | 初回受信の固定fixture成功。公開相互運用とDocker照合は未実施 |

## Webmentionの検証境界

fixtureは初回フォーム受付、targetリンク、相対参照、投稿資格を確認する。
再通知の試験は取得ゼロと保存済みAtomの不変性を確認する。
リレーはmf2必須のHTML受信に範囲を限定する。
リレーはWebmentionの更新・削除追従に対応しない。
検証者は包括的なWebmention準拠を主張しない。
公開配備とwebmention.rocksの受信試験は未実施とする。
仕様の参照先: https://www.w3.org/TR/webmention/

## 実行検証の引き継ぎ

Nawashiroはコンテナの実行検証を担当する。
本リポジトリは実行確認まで該当タスクを未完了として保持する。

## 未検証事項

`docker info` はDocker daemonへの接続不能を返した。
検証者はDockerホスト上の次の実行を未完了として残す。

- Docker build、単一コンテナ起動、非root UID 65532の権限確認。
- ホストディレクトリを保持したコンテナ停止・再作成。
- 運用文書のバックアップ・復元コマンド。
- 公開環境のWebmention相互運用。

検証者は未検証タスクへチェックを付けない。
検証者はOpenSpec変更をアーカイブしない。
