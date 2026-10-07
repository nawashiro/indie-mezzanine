# 検証記録

## 実行環境

検証者は Linux amd64 と Go 1.27.1 を使った。
検証者は全ての実装を直接実施した。
中断したサブエージェントは実装ファイルを変更しなかった。
検証者は実際の原本記事をfixtureへ固定しない。

## 成功した検査

検証者は次の検査を実行した。

```sh
go mod verify
go vet ./...
go test -race -count=3 ./...
CGO_ENABLED=0 go build -trimpath -o /opt/data/cache/mezzanine ./cmd/mezzanine
openspec validate build-public-collection-relay --strict
```

Goの依存検査は `all modules verified` を返した。
Goの競合検査は全パッケージで成功した。
OpenSpecは `Change 'build-public-collection-relay' is valid` を返した。

検証者は `govulncheck v1.7.0` を実行した。
最初のx/net v0.39.0はHTML解析経路に既知の脆弱性を持った。
検証者はx/net v0.56.0へ更新した。
更新後の検査は `No vulnerabilities found.` を返した。

検証者はDocker Compose v2.39.4の配布チェックサムを確認した。
次の設定検査は終了コード0を返した。

```sh
PUBLIC_URL=https://relay.example/ docker-compose -f compose.yml config --quiet
```

検証者はhadolint v2.12.0の配布チェックサムを確認した。
`hadolint Dockerfile` は終了コード0を返した。
検証者は両FROMイメージのレジストリmanifestを取得した。
両manifestはHTTP 200を返した。
Dockerfileは取得したdigestでイメージを固定した。
これらの静的検査はコンテナ実行の代わりにならない。

## 実バイナリの確認

検証者は非CGOバイナリをUID 1000で起動した。
検証者は一時保存先とループバックの一時ポートを使った。

- ルートHTTP: 200。
- WebmentionのLink広告: 設定PUBLIC_URL由来のendpoint。
- healthcheckコマンド: 終了コード0。
- 通知フォーム受付: HTTP 202。
- SIGTERM後の停止: 終了コード0。
- SQLiteの読み戻し: 受付jobを一件保持。

この確認はコンテナUID 65532の動作確認ではない。

## タスク別の証拠

| タスク | 状態 | 主な検査 |
| --- | --- | --- |
| 1.1 | 完了 | 固定Go環境の非CGOビルド、実バイナリ起動 |
| 1.2 | 完了 | go mod verify、依存一覧、配布ライセンス原文 |
| 1.3 | 完了 | TestStorePersistenceAndTransactions、非CGOビルド |
| 2.1 | 完了 | TestSafeFetcherRejects、TestConfig |
| 2.2 | 完了 | TestRedirectChecks、proxy無効化検査 |
| 2.3 | 完了 | TestBodyAndTimeLimitsのgzip、chunked、遅延 |
| 2.4 | 完了 | TestFetchDNSChangeIsolated、go test -race |
| 3.1 | 完了 | TestLinkAttributes、TestParseConstraints、最終URLの相対参照 |
| 3.2 | 完了 | TestCollection、TestParseConstraints |
| 3.3 | 完了 | TestMF2StructureAndEquivalentHash、JSON往復とHTML除去 |
| 3.4 | 完了 | examples/post.html、TestExampleAndDocumentCommands |
| 4.1 | 完了 | TestStoreSchemaAndUnavailable、一意制約、再接続 |
| 4.2 | 完了 | TestStorePersistenceAndTransactionsの同値・移動・rollback |
| 4.3 | 完了 | TestQueue、TestWorkerRetriesFinite、TestBackupRestoreAndInterruptedJobs |
| 4.4 | 完了 | 保存境界の説明、mf2読み戻し、復旧試験 |
| 5.1 | 完了 | TestHTTPAdmission、広告、400/405/202/503 |
| 5.2 | 完了 | TestProcessOutcomes、TestE2EAndRestart |
| 5.3 | 完了 | TestConfig、TestHealthAndShutdown、実バイナリの正常停止 |
| 5.4 | 完了 | 文書のcurlを実行。202時点の未掲載とworker後の掲載を区別 |
| 6.1 | 完了 | TestAtom、TestDateFallback、XML解析、text、メタデータ |
| 6.2 | 完了 | TestFeedOrderingLimitAndTimes、TestE2EAndRestartの404と空feed |
| 6.3 | 完了 | 投稿・購読説明、文書の購読経路のHTTP取得 |
| 7.1 | 未完了 | Dockerfileとcompose.yml作成、静的検査成功。Docker実行なし |
| 7.2 | 未完了 | 運用説明作成、Goのバックアップ復元成功。コンテナ運用未実行 |
| 8.1 | 完了 | TestE2EAndRestart、固定ローカルsourceとテスト専用Fetcher |
| 8.2 | 未完了 | Store/App再起動の検証成功。コンテナ停止・再作成なし |
| 8.3 | 完了 | go vet、go test -race、OpenSpec strict |
| 8.4 | 未完了 | 仕様由来のfixture成功。公開相互運用とDocker結果の照合なし |

## Webmentionの検証範囲

検証者は [W3C Webmention Recommendation](https://www.w3.org/TR/webmention/) を参照した。
fixtureはフォーム受付、対象属性、正確なtargetリンク、相対参照、再通知、撤回を検査する。
リレーはmf2必須のHTML受信に範囲を限定する。
リレーは一般のWebmention送信クライアントを提供しない。
公開URLへの配備と [webmention.rocks](https://webmention.rocks/) の受信試験は未実施とする。
検証者は包括的なWebmention準拠を主張しない。

## 未検証事項

現環境はDocker daemonとDocker socketを持たない。
`docker info` はdaemon接続不能を返した。
検証者はユーザーの了承に従い、Goの実行検証とDocker成果物の作成まで進めた。

次の検証はDockerホスト上で実行する。

- Docker buildと単一コンテナ起動。
- UID 65532の書込権限とヘルス確認。
- 永続ボリューム付きの停止・再作成。
- 未処理通知と既存投稿の再処理・重複防止。
- 運用文書のバックアップ・復元コマンド。
- 公開環境のWebmention相互運用。

検証者は未検証タスクのチェックを付けない。
検証者はOpenSpec変更をアーカイブしない。
