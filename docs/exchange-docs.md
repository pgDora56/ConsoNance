# Discord Audio Stream Bot -> ConsoNance
###### tags: `Introquiz`, `system`

## おねがい

今回の開発やAzureサーバーの運営に必要な費用を以下のpixivFANBOXで募っています。

{%preview https://dora417.fanbox.cc %}

多分これまで支援してくださっていた方のクレカの設定が切れちゃったとかというのもあるかなと思うんですが、ゆるやかに支援者が減っており、**経済的にピンチなサービスになってきております。**

ご余裕がありましたら、100円からでもご支援いただけると幸いです＆「継続してるよ！！」という方、ぜひクレカの設定などが切れていないかご確認お願いします🙇


<br />
<br />
<br />
<br />
<br />


## 本編

この `Consonance` はこれまでのDiscord Audio Stream Botの代替となるツールです。PC音をDiscordの通話にステレオで載せることができます。

Javaへの依存がないので、色々なアップデートへの強さがある点が一番のメリットになります。あとは起動がちょっぴり簡単。

ここではWindowsでの例を挙げますが、開発自体はWindows, Mac, Linuxのいずれでも動くようなものを目指しましたので、多分どのOSでもできると思います。

### ダウンロード

以下のページからダウンロードできます。最新の `ConsoNance-xx.xx.zip`（xには数字が入る） というものをダウンロードしてください。

https://github.com/pgDora56/ConsoNance/releases

### Bot設定の変更

Discord Developer Portalから、 `Server Members Intent` と `Message Content Intent` をOnにする設定が必要です。

https://discord.com/developers/applications

上記からいつも使っているBotを選んで以下の箇所をOnにしてください。

![image](https://hackmd.io/_uploads/SktTi6LQZe.png)


### 最初の起動

ダウンロードしたら、そのZIPファイルを展開してください。
`consonance-win.exe` というファイルが出てくると思います。

それを適当なフォルダにおいてください。設定ファイルやログファイルが同じフォルダに出てきたりしますので、他にファイルがないフォルダが良いかと思います。

適当なフォルダに置けたら実行してみましょう。

> このとき「WindowsによってPCが保護されました」というような青い画面が出てくるかもしれません。その場合は以下のFMVのサイトなどを参考に、「実行」を押してください。
> https://www.fmworld.net/cs/azbyclub/qanavi/jsp/qacontents.jsp?PID=0209-8188
> 
> 多分1回やればでなくなるはずです。


実行すると以下のような黒い画面がでてきます。怖くないので安心してください。

![image-01](https://hackmd.io/_uploads/B1g1WRSmWx.jpg)

この画面が出ていればひとまず実行成功です。

### Tokenを入れる

先程の黒い画面の一番下を見ると「Discord bot tokenをここに貼り付けろ」と英語で言われています。そのとおりに、Discord Audio Stream Botで使っていたものと同じtokenをここに貼り付けてください。

Discord Audio Stream BotのTokenは以下の箇所からコピーするのが楽だと思います。
![image](https://hackmd.io/_uploads/HyUYrArX-e.png)

> 何らかの理由でコピーできない場合は、Discord Audio Stream Botと同じフォルダに `config.json` というファイルがあるので、それをメモ帳などで開いて `botToken` と書いてあるところからも見られます。 左右の`"` を入れないようにコピーして下さい。
> 
> それでもダメな場合は、Discord Developer PortalからTokenの再発行を試してください（既存のトークンは使えなくなります）。

### Audio devicesを選ぶ

Tokenを入れられたらDiscordに何の音を選ぶのかを聞かれます。下の例だと `[1]` から `[5]` とあるかなと思います。`ライン（Yamaha SYNCROOM Driver（WDM））`を使っている人が多いでしょうか。こちらもこれまで使っていたAudio devicesの数字だけを入力してEnterを押してみてください。

![image-02](https://hackmd.io/_uploads/ByQkWCBXbl.jpg)

入れられると、以下のように「今回選んだデバイスをデフォルトのデバイスにしますか？」と聞かれます。末尾に `(y/n)`とあるのは yes or no ということです。

使うたびにデバイスを変える方はあまり多くないと思いますので、一旦yesとするとよいかと思います。 `y` とだけいれてEnterを押してみましょう（毎回変える方は `n` と入れてくださいね）。

![image-03](https://hackmd.io/_uploads/ryv1WAB7Wl.jpg)

### 成功

以下のようなメッセージがでてきたら起動成功です。

![image-04](https://hackmd.io/_uploads/rJePJWRrQ-l.jpg)


呼び出したいサーバーで`@bot join #channel-name` とするとチャンネルに入ってきます。
つまり、 `Mocho` というbotを `ラウンジ` チャンネルに呼びたければ `@Mocho join #ラウンジ` と入力します。

![image](https://hackmd.io/_uploads/HJMow0SmZx.png)

このときの注意点が
- Discord Audio Stream Botのときのような `/join` は使わない
    - 上に出てくるポップアップから呼ぶのに慣れてきたかと思いますが、そんな上等な機能がついていません。
- #のあとに**ボイスチャンネル**の名前を入れること
    - テキストチャンネルのほうが優先して出てきがちなので間違えないようにしてください。


あとはいつも通りです。Botを終了するときは、黒い画面の右上の×ボタンを押すと終了します。


### 失敗例

失敗すると以下のような感じになったりします（最後の行に `Press Enter to exit...` と出てくる）。

![image-e01](https://hackmd.io/_uploads/SkDkbRB7-x.jpg)

これはDiscord Bot Tokenが間違っている例です。一番出やすいのはこれだと思います。

このままやると、以下のように永遠に起動したらエラーが出続けてしまいます。

![image-e02](https://hackmd.io/_uploads/S1ePyWABX-e.jpg)

そんなときは、 `consonance-win.exe` と同じフォルダにできているはずの `config.yaml` というファイルを一旦削除してみてください。これで初期状態に戻りますので、再度上の手順などを試してください。
何か困ったらこれで常に初期状態に戻せますので、困ったときの再実行に試してください。

### 2回目以降の起動

2回目以降は、 `consonance-win.exe` を実行するだけで、以下のような画面になり、すぐに起動・利用していくことができます。

![image-s01](https://hackmd.io/_uploads/HJP1bRrmZx.jpg)


### おわりに 

Discord Audio Stream Botはいまも変わらず大変便利なんですが、Javaへの依存が結構あったりしたので、それ対策も兼ねて今回はこちらを作ってみました。まあソフトウェアというもの自体が何らかの上で動くものなので、完璧に対策できることはないんですが、Windowsの大幅な改変とかそういうことがない限りはちゃんと動き続けるんじゃないかなと思っています。

加えて、Botを自分で作って自由度が大変上がったのも良かったポイント。ぜひつかっていただいて、なにか「ここをこうしてほしい！」みたいなものがあったら気軽に声をかけてください～
