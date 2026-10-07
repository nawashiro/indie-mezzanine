# 公開リレーの運用

## 起動

運用者は Docker Engine と Docker Compose を用意する。
運用者は `.env.example` を `.env` へコピーする。
運用者は `PUBLIC_URL` を実際の公開ルートURLへ変更する。
公開URLは認証情報、query、fragment、サブパスを含まない。

```sh
cp .env.example .env
export HOST_DATA_DIR=./data
sudo install -d -o 65532 -g 65532 -m 0750 "$HOST_DATA_DIR"
docker compose config
docker compose up -d --build
docker compose ps
docker compose exec relay /mezzanine healthcheck
```

運用者は `HOST_DATA_DIR` と `.env` の保存先を同じ値に揃える。
Composeはホストの保存先をコンテナの `/data` へバインドマウントする。
Composeは存在しないホストディレクトリを自動作成しない。
コンテナは UID/GID 65532 で動く。
運用者はそのユーザーだけに必要な書き込み権限を与える。
アプリは書き込み不能なら起動に失敗する。
SELinuxを使うホストでは、運用者は保存先のコンテナ用ラベルも設定する。

ComposeはHTTPをループバックの8080番へ公開する。
運用者は既存のリバースプロキシで公開HTTPSを終端する。
Composeは外部DBと外部キューを起動しない。
Dockerfileは匿名VOLUMEを宣言しない。

## 再作成と中断復旧

```sh
docker compose up -d --force-recreate
docker compose exec relay /mezzanine healthcheck
```

運用者は同じホストディレクトリを再作成後も指定する。
コンテナの削除はホストディレクトリを削除しない。
**運用者は保存先を削除しない。**
workerは未掲載の中断jobを有限回再処理する。
保存済みsourceの中断jobは取得なしで完了する。
同じsourceは同時に処理しない。
source URL単独の一意制約は重複掲載を防ぐ。
再通知はjobを増やさず、固定スナップショットを変更しない。

## バックアップと復元

**バックアップ前に、運用者はリレーを停止する。**
運用者はSQLite本体とWAL関連ファイルを同じ時点で保存する。
運用者はバックアップをリポジトリへコミットしない。
次の例は `backup/mezzanine.tar` を置き換える。
運用者は必要な旧バックアップを先に退避する。

```sh
docker compose stop relay
mkdir -p backup
sudo tar -C "$HOST_DATA_DIR" -cpf "$PWD/backup/mezzanine.tar" .
docker compose start relay
```

**復元前に、運用者はリレーを停止する。**
運用者は互換スキーマのバックアップだけを使う。
次の例は現データを別ディレクトリへ退避する。
運用者は復元先に古いWALを混在させない。
`.before-restore` が存在する場合、運用者は先に別の退避先を選ぶ。

```sh
set -e
docker compose stop relay
test ! -e "${HOST_DATA_DIR}.before-restore"
sudo mv -- "$HOST_DATA_DIR" "${HOST_DATA_DIR}.before-restore"
sudo install -d -o 65532 -g 65532 -m 0750 "$HOST_DATA_DIR"
sudo tar -C "$HOST_DATA_DIR" -xpf "$PWD/backup/mezzanine.tar"
sudo chown -R 65532:65532 "$HOST_DATA_DIR"
docker compose start relay
```

運用者は新しいシェルでも `HOST_DATA_DIR` を同じ値へ設定する。
運用者は復元後のヘルスと購読feedを確認する。
保存済みsourceは復元後も再取得しない。
アプリは未掲載の通知だけを取得・検証する。
現環境ではDocker daemonがないため、Dockerコマンドの実検証は未完了とする。

## スキーマの互換性

本改訂のスキーマ版は2とする。
アプリは旧仕様の版1を拒否し、自動変換しない。
運用者は旧検証データを保全し、新規ディレクトリで本改訂を確認する。
不可逆なスキーマ移行は本変更の対象外とする。

## 設定と資源上限

| 環境変数 | 既定値 | 意味 |
| --- | --- | --- |
| PUBLIC_URL | 必須 | 受付targetの公開ルートURL |
| DATA_DIR | 必須。コンテナ内は /data | SQLite保存先 |
| HOST_DATA_DIR | ./data | Composeのホスト側バインド先。アプリは参照しない |
| LISTEN_ADDR | :8080 | HTTP待受 |
| HEALTH_URL | http://127.0.0.1:8080/healthz | healthcheckコマンドの取得先。待受変更時に指定 |
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
