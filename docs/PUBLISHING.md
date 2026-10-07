# 投稿と購読

投稿者は `examples/post.html` を公開ページの出発点にする。
投稿者は一つのページに一つのh-entryを置く。
投稿者は一つのUUID URNをrel=collectionで宣言する。
投稿者は受付targetへのリンクを同じHTMLに置く。
投稿者はHTMLをHTTPまたはHTTPSの公開URLへ配置する。
公開sourceは80番または443番を使う。
リレーは内部IP、URL内認証情報、その他のポートを拒否する。

```sh
SOURCE=https://author.example/post
PUBLIC_URL=https://relay.example/
WEBMENTION_ENDPOINT="${PUBLIC_URL}webmention"
```

投稿者は次のフォームで公開ページを通知する。

```sh
curl --fail-with-body -i \
  --data-urlencode "source=$SOURCE" \
  --data-urlencode "target=$PUBLIC_URL" \
  "$WEBMENTION_ENDPOINT"
```

HTTP 202は永続受付だけを示す。
workerは原本のリンクとmf2を確認してから掲載する。
掲載後、購読者は次のURLをAtomリーダーへ登録する。

```text
https://relay.example/collections/550e8400-e29b-41d4-a716-446655440000.atom
```

## スナップショットと再通知

リレーはsource URLごとの最初の検証成功結果を一件だけ保存する。
スナップショットは通知後の取得・検証時点の内容になる。
初回失敗後の通知は、最初の検証成功まで掲載を試みる。
保存済みsourceの有効な再通知はHTTP 200になる。
リレーは再通知で原本を取得せず、新しいjobを作らない。
リレーは最初の本文、所属、識別子、日時を保持する。

リレーは原本の編集・所属変更・削除に追従しない。
再通知は撤回の手段にならない。
リレーはWebmentionの更新・削除追従には対応しない。
リレーは定期巡回と通知ごとの履歴追加を行わない。
別のsource URLは別の投稿になる。
リレーは異なるURLの同一投稿を推測しない。

source URLは投稿のAtom idになる。
リダイレクトとu-urlはそのidを変えない。

## 保存とAtom

mf2はHTML表現を全階層から除去する。
保存データはh-card、children、content.valueを保持する。
Atomはcontent.valueをtype=textで出力する。
パーサーの段落境界は元HTMLと異なる場合がある。
リレーは段落を再構成しない。

feed idと既定タイトルは正規UUID URNになる。
entry titleはnameを使う。
name欠落時、entry titleはsource URLになる。
著者名欠落時、entry authorは「投稿者不明」になる。
publishedは有効なISO日時だけを出す。
updatedは初回スナップショットの有効なISO日時か初回保存日時を使う。
entryとfeedのupdatedは再通知で変わらない。
feed.updatedは別sourceの新規掲載でだけ進む。
日時はRFC3339、時差なしISO日時、ISO日付を受理する。
時差なし日時と日付はUTCとして扱う。
同時刻のentryはsource URLの昇順になる。
未知のcollectionと不正なUUIDは404になる。

## 安全な取得の検証境界

本番はsafeurlの接続時IP検査を使う。
各redirectはscheme、port、認証情報、回数を検査する。
本番は環境proxyとsafeurlテストモードを使わない。
本番は著者URLや画像を補助取得しない。

E2Eはテスト専用Fetcherで固定ローカルsourceへ接続する。
本文・時間テストはローカルtransportを注入する。
これらの注入は `_test.go` にだけ存在する。
IP拒否テストは本番transportを使う。
DNS変更テストは独立プロセスの局所resolverを使う。
DNS試験はsafeurlの競合するテスト専用tracerを設置しない。
DNS試験はsafeurlのresolverと接続時IP検査を維持する。
