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
	"github.com/gen2brain/malgo"
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
		DiscordToken    string `yaml:"discord_token"`
		ChannelID       string `yaml:"channel_id"`
		GuildID         string `yaml:"guild_id"`
		AudioDeviceName string `yaml:"audio_device_name"`
		ListDevices     bool   `yaml:"list_devices"`
	}

	var cfg Config
	decoder := yaml.NewDecoder(configFile)
	if err := decoder.Decode(&cfg); err != nil {
		log.Fatalf("Failed to parse config.yaml: %v", err)
	}

	// デバイス一覧表示モード
	if cfg.ListDevices {
		if err := listAudioDevices(); err != nil {
			log.Fatalf("Failed to list audio devices: %v", err)
		}
		return
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

	// システムオーディオのストリーミング開始
	// （バックグラウンドで実行）
	go func() {
		if err := streamSystemAudio(voiceConnection, cfg.AudioDeviceName); err != nil {
			log.Fatalf("Failed to stream system audio: %v", err)
		}
	}()

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

// streamSystemAudio captures system audio (loopback) and streams it to Discord
func streamSystemAudio(v *discordgo.VoiceConnection, deviceName string) error {
	// VoiceConnectionがReadyであることを確認
	if !v.Ready {
		return fmt.Errorf("voice connection is not ready")
	}

	const (
		sampleRate = 48000
		channels   = 2
		frameSize  = 960 // 20ms at 48kHz
	)

	// Opusエンコーダーの作成
	encoder, err := gopus.NewEncoder(sampleRate, channels, gopus.Audio)
	if err != nil {
		return fmt.Errorf("failed to create opus encoder: %v", err)
	}

	// malgoコンテキストの初期化
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return fmt.Errorf("failed to initialize malgo context: %v", err)
	}
	defer func() {
		_ = ctx.Uninit()
		ctx.Free()
	}()

	// デバイスコンフィグの設定
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Loopback)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = uint32(channels)
	deviceConfig.SampleRate = uint32(sampleRate)
	deviceConfig.Alsa.NoMMap = 1

	// デバイス名が指定されている場合、そのデバイスを探す
	if deviceName != "" {
		deviceInfo, err := findDeviceByName(ctx, deviceName)
		if err != nil {
			return fmt.Errorf("failed to find device '%s': %v", deviceName, err)
		}
		deviceConfig.Capture.DeviceID = deviceInfo.ID.Pointer()
		log.Printf("Using audio device: %s", deviceName)
	} else {
		log.Println("Using default loopback device")
	}

	// オーディオバッファ（PCMデータを蓄積）
	pcmBuffer := make([]int16, 0, frameSize*channels*2)

	// Speaking状態を設定
	if err := v.Speaking(true); err != nil {
		return fmt.Errorf("failed to set speaking state: %v", err)
	}
	defer v.Speaking(false)

	log.Println("Starting system audio capture (loopback mode)...")

	// データコールバック：音声データが取得されるたびに呼ばれる
	var captureCallbacks = malgo.DeviceCallbacks{
		Data: func(pOutputSample, pInputSamples []byte, framecount uint32) {
			// バイト列をint16スライスに変換
			samples := make([]int16, len(pInputSamples)/2)
			for i := 0; i < len(samples); i++ {
				samples[i] = int16(pInputSamples[i*2]) | int16(pInputSamples[i*2+1])<<8
			}

			// バッファに追加
			pcmBuffer = append(pcmBuffer, samples...)

			// バッファが1フレーム分以上溜まったら送信
			for len(pcmBuffer) >= frameSize*channels {
				// 1フレーム分を取り出す
				frame := pcmBuffer[:frameSize*channels]
				pcmBuffer = pcmBuffer[frameSize*channels:]

				// Opusエンコード
				opusData, err := encoder.Encode(frame, frameSize, 1000)
				if err != nil {
					log.Printf("Failed to encode audio: %v", err)
					continue
				}

				// Discordに送信（ノンブロッキング）
				select {
				case v.OpusSend <- opusData:
				default:
					// チャンネルがいっぱいの場合はスキップ
					log.Println("Warning: OpusSend channel full, skipping frame")
				}
			}
		},
	}

	// デバイスの初期化と開始
	device, err := malgo.InitDevice(ctx.Context, deviceConfig, captureCallbacks)
	if err != nil {
		return fmt.Errorf("failed to initialize capture device: %v", err)
	}

	if err := device.Start(); err != nil {
		device.Uninit()
		return fmt.Errorf("failed to start capture device: %v", err)
	}

	log.Println("System audio streaming started! Press CTRL+C to stop.")

	// シグナル待機（main関数で行うため、ここでは無限ループ）
	// この関数はバックグラウンドで動き続ける
	select {}
}

// listAudioDevices lists all available audio devices
func listAudioDevices() error {
	// malgoコンテキストの初期化
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return fmt.Errorf("failed to initialize malgo context: %v", err)
	}
	defer func() {
		_ = ctx.Uninit()
		ctx.Free()
	}()

	// 再生デバイス（ループバックに使用可能）の取得
	infos, err := ctx.Devices(malgo.Playback)
	if err != nil {
		return fmt.Errorf("failed to get playback devices: %v", err)
	}

	fmt.Println("\n=== Available Audio Devices (Playback) ===")
	fmt.Println("These devices can be used for loopback capture")
	fmt.Println()

	if len(infos) == 0 {
		fmt.Println("No playback devices found.")
	} else {
		for i, info := range infos {
			fmt.Printf("[%d] %s\n", i+1, info.Name())
			fmt.Printf("    ID: %v\n", info.ID)
			if info.IsDefault > 0 {
				fmt.Println("    (Default Device)")
			}
			fmt.Println()
		}
	}

	fmt.Println("\nTo use a specific device, set 'audio_device_name' in config.yaml")
	fmt.Println("Example: audio_device_name: \"Speakers (Realtek High Definition Audio)\"")

	return nil
}

// findDeviceByName finds a device by its name
func findDeviceByName(ctx *malgo.AllocatedContext, deviceName string) (*malgo.DeviceInfo, error) {
	// 再生デバイスから検索
	infos, err := ctx.Devices(malgo.Playback)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %v", err)
	}

	for _, info := range infos {
		if info.Name() == deviceName {
			return &info, nil
		}
	}

	return nil, fmt.Errorf("device not found: %s", deviceName)
}
