package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/yaml.v3"
	"layeh.com/gopus"
)

func main() {
	// config.yamlの読み込み
	configFile, err := os.Open("config.yaml")
	if err != nil {
		log.Fatalf("Failed to open config.yaml: %v", err)
	}
	defer configFile.Close()

	// 設定ファイルの構造体定義
	type Config struct {
		DiscordToken string `yaml:"discord_token"`
		ChannelID    string `yaml:"channel_id"`
		GuildID      string `yaml:"guild_id"`
	}

	var cfg Config
	decoder := yaml.NewDecoder(configFile)
	if err := decoder.Decode(&cfg); err != nil {
		log.Fatalf("Failed to parse config.yaml: %v", err)
	}

	// Discordセッションの作成
	discord, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.Fatalf("Failed to create Discord session: %v", err)
	}

	// Intentの設定（ボイスステートの取得に必要）
	discord.Identify.Intents = discordgo.IntentsGuildVoiceStates | discordgo.IntentsGuilds

	// Discordセッションのオープン
	if err := discord.Open(); err != nil {
		log.Fatalf("Failed to open Discord session: %v", err)
	}
	defer discord.Close()

	log.Println("Bot is now running. Connecting to voice channel...")

	// ボイスチャンネルへの接続
	voiceConnection, err := discord.ChannelVoiceJoin(cfg.GuildID, cfg.ChannelID, false, true)
	if err != nil {
		log.Fatalf("Failed to join voice channel: %v", err)
	}
	defer voiceConnection.Disconnect()

	// VoiceConnectionがReadyになるまで待機
	log.Println("Waiting for voice connection to be ready...")
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	ready := false
	for !ready {
		select {
		case <-timeout:
			log.Println("Warning: Timeout waiting for voice connection to be ready, proceeding anyway...")
			ready = true
		case <-ticker.C:
			if voiceConnection.Ready {
				ready = true
				log.Println("Voice connection is ready!")
			}
		}
	}

	fmt.Printf("Successfully connected to voice channel: %s\n", cfg.ChannelID)

	// ビープ音を再生
	log.Println("Playing beep sound...")
	if err := playBeep(voiceConnection); err != nil {
		log.Printf("Failed to play beep: %v", err)
	} else {
		log.Println("Beep sound finished!")
	}

	// プログラムの終了を待機（Ctrl+Cで終了）
	log.Println("Bot is running. Press CTRL+C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("Bot is shutting down...")
}

// playBeep generates and plays a simple beep sound
func playBeep(v *discordgo.VoiceConnection) error {
	// VoiceConnectionがReadyであることを再確認
	if !v.Ready {
		return fmt.Errorf("voice connection is not ready")
	}

	// Opusエンコーダーの作成
	// 48kHz, 2チャンネル（ステレオ）
	const (
		sampleRate = 48000
		channels   = 2
		frameSize  = 960 // 20msのフレーム
		frequency  = 440 // A4の音（ラの音）
		duration   = 1.0 // 1秒間
	)

	encoder, err := gopus.NewEncoder(sampleRate, channels, gopus.Audio)
	if err != nil {
		return fmt.Errorf("failed to create opus encoder: %v", err)
	}

	// ビープ音の生成とエンコード
	totalSamples := int(sampleRate * duration)
	pcm := make([]int16, frameSize*channels)

	// Speaking状態を設定
	if err := v.Speaking(true); err != nil {
		return fmt.Errorf("failed to set speaking state: %v", err)
	}
	defer v.Speaking(false)

	// 少し待機してOpusSendチャンネルが準備完了するのを待つ
	time.Sleep(100 * time.Millisecond)

	// フレームごとの送信タイミングを管理
	frameDuration := time.Duration(frameSize) * time.Second / time.Duration(sampleRate)

	for sample := 0; sample < totalSamples; sample += frameSize {
		start := time.Now()

		// PCMデータの生成（サイン波）
		for i := 0; i < frameSize; i++ {
			if sample+i >= totalSamples {
				break
			}
			// サイン波を生成（440Hz）
			value := math.Sin(2.0 * math.Pi * frequency * float64(sample+i) / float64(sampleRate))
			// 振幅を調整（音量を小さめに）
			pcmValue := int16(value * 0.3 * 32767)

			// ステレオなので両チャンネルに同じ値を設定
			pcm[i*channels] = pcmValue
			pcm[i*channels+1] = pcmValue
		}

		// OpusにエンコードしてVoiceConnectionに送信
		// gopusは[]int16を直接受け取る
		opusData, err := encoder.Encode(pcm, frameSize, 1000)
		if err != nil {
			return fmt.Errorf("failed to encode: %v", err)
		}

		v.OpusSend <- opusData

		// 次のフレームまで適切な時間待機（20ms）
		elapsed := time.Since(start)
		if elapsed < frameDuration {
			time.Sleep(frameDuration - elapsed)
		}
	}

	return nil
}
