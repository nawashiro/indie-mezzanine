# 公開リレーの運用

## 起動

運用者は Docker Engine と Docker Compose を用意する。
運用者は `.env.example` を `.env` へコピーする。
運用者は `PUBLIC_URL` を実際の公開ルートURLへ変更する。
公開URLは HTTP または HTTPS を使う。
公開URLは末尾の `/` を含む。
公開URLは認証情報、query、fragment、サブパスを含まない。

```sh
cp .env.example .env
docker compose config
docker compose up -d --build
docker compose ps
docker compose exec relay /mezzanine healthcheck
```

Compose は HTTP をループバックの8080番へ公開する。
運用者は既存のリバースプロキシで公開HTTPSを終端する。
Compose は外部DBや外部キューを起動しない。
コンテナは UID/GID 65532 で動く。
名前付きボリュームは `/data` を保持する。
バインドマウントを使う場合、運用者はディレクトリの書込権限を UID 65532 へ与える。

## 再作成

```sh
docker compose up -d --force-recreate
docker compose exec relay /mezzanine healthcheck
```

**データを保持する場合、運用者は `docker compose down -v` を実行しない。**
worker は起動時に中断jobをpendingへ戻す。
試行上限に達した中断jobはfailedになる。
同じsourceは同時に処理しない。
同じsource/targetの掲載は一件になる。

## バックアップと復元

**バックアップ前に、運用者はリレーを停止する。**
運用者はSQLite本体とWAL関連ファイルを同じ時点でコピーする。
運用者はバックアップをリポジトリへコミットしない。

```sh
docker compose stop relay
mkdir -p backup
docker compose cp relay:/data/. ./backup/
docker compose start relay
```

**復元前に、運用者はリレーを停止する。**
運用者は互換スキーマのバックアップだけを復元する。
次の例は保存先を空にしてから全ファイルを復元する。
運用者は削除前に現在データを別のバックアップへ保存する。

運用者はDockerホスト上で補助コンテナを使ってボリュームを復元する。

```sh
docker compose stop relay
docker run --rm --user 0 \
  -v mezzanine_data:/data -v "$PWD/backup:/backup:ro" alpine:3.22 \
  sh -c 'rm -f /data/mezzanine.db /data/mezzanine.db-wal /data/mezzanine.db-shm && cp -a /backup/. /data/ && chown -R 65532:65532 /data'
docker compose start relay
```

Composeのプロジェクト名は `mezzanine` とする。
運用者がプロジェクト名を変更した場合、運用者はボリューム名も変更する。
現環境はDocker daemonを持たないため、コンテナ関連コマンドは未実行とする。

## 設定と資源上限

| 環境変数 | 既定値 | 意味 |
| --- | --- | --- |
| PUBLIC_URL | 必須 | 受付targetの公開ルートURL |
| DATA_DIR | 必須。コンテナ内は /data | SQLite保存先 |
| LISTEN_ADDR | :8080 | HTTP待受 |
| FETCH_TIMEOUT | 10s | ヘッダーと本文を含む取得時間 |
| MAX_BODY_BYTES | 1048576 | 展開後本文上限 |
| MAX_FORM_BYTES | 16384 | 受付フォーム上限 |
| MAX_REDIRECTS | 5 | 転送回数 |
| WORKERS | 2 | 同時処理数 |
| QUEUE_LIMIT | 1000 | pendingとrunningの合計上限 |
| FEED_LIMIT | 100 | feedの投稿上限 |
| MAX_ATTEMPTS | 3 | 中断を含む試行上限 |
| JOB_RETENTION | 168h | jobの保持期間 |

設定検査は0以下を拒否する。
workerは一時失敗を有限回再試行する。
処理済みjobと古いpendingは保持期限後の定期掃除で消える。
workerの停止とDBの利用不能はヘルス確認を失敗させる。
原本HTMLと取得エラーのURLは永続ログへ出力しない。
SQLiteはHTML除去済みmf2、collection、投稿識別子、日時、job状態を保持する。
