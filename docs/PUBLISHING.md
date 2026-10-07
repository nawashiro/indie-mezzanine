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

## 更新と撤回

編集後、投稿者は同じsourceとtargetを再通知する。
source URLは投稿のAtom idになる。
リダイレクトとu-urlはそのidを変えない。
同値再通知は更新日時を進めない。
所属変更は旧collectionから撤回し、新collectionへ掲載する。

削除時、投稿者は404または410を返してから再通知する。
targetリンクや投稿資格の消失も撤回になる。
タイムアウト、5xx、安全拒否、本文上限超過は既存投稿を残す。
リレーは原本を定期巡回しない。
更新と削除は再通知まで反映しない。

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
updatedは有効なISO日時か内容変更の反映日時を使う。
日時はRFC3339、時差なしISO日時、ISO日付を受理する。
時差なし日時と日付はUTCとして扱う。
同時刻のentryはsource URLの昇順になる。
既知の空collectionは200で空feedを返す。
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
