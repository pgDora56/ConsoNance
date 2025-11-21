// ConsoNance - Audio Stream Bot for Discord
// Copyright (C) 2025 Kazuki F.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gen2brain/malgo"
	"gopkg.in/yaml.v3"
	"layeh.com/gopus"
)

// Bot state management
type BotState struct {
	sync.RWMutex
	voiceConnection *discordgo.VoiceConnection
	guildID         string
	channelID       string
	audioDeviceName string
	isStreaming     bool
	stopStreaming   chan bool
}

var (
	botState *BotState
	config   *Config
	session  *discordgo.Session
)

// Config structure
type Config struct {
	DiscordToken    string `yaml:"discord_token"`
	ChannelID       string `yaml:"channel_id"`
	GuildID         string `yaml:"guild_id"`
	AudioDeviceName string `yaml:"audio_device_name"`
	ListDevices     bool   `yaml:"list_devices"`
}

func main() {
	// config.yamlの読み込み
	configFile, err := os.Open("config.yaml")
	if err != nil {
		log.Fatalf("Failed to open config.yaml: %v", err)
	}
	defer configFile.Close()

	config = &Config{}
	decoder := yaml.NewDecoder(configFile)
	if err := decoder.Decode(config); err != nil {
		log.Fatalf("Failed to parse config.yaml: %v", err)
	}

	// デバイス一覧表示モード
	if config.ListDevices {
		if err := listAudioDevices(); err != nil {
			log.Fatalf("Failed to list audio devices: %v", err)
		}
		return
	}

	// BotStateの初期化
	botState = &BotState{
		guildID:         config.GuildID,
		audioDeviceName: config.AudioDeviceName,
		stopStreaming:   make(chan bool),
	}

	// Discordセッションの作成
	session, err = discordgo.New("Bot " + config.DiscordToken)
	if err != nil {
		log.Fatalf("Failed to create Discord session: %v", err)
	}

	// Intentの設定
	session.Identify.Intents = discordgo.IntentsGuildVoiceStates | discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	// メッセージハンドラの登録
	session.AddHandler(messageCreate)

	// Discordセッションのオープン
	if err := session.Open(); err != nil {
		log.Fatalf("Failed to open Discord session: %v", err)
	}
	defer session.Close()

	log.Println("Bot is now running. Mention me with commands!")
	log.Println("Commands: @Bot join #channel-name, @Bot leave, @Bot status, @Bot help")

	// config.yamlにチャンネルIDが指定されていたら自動接続
	if config.ChannelID != "" {
		log.Printf("Auto-connecting to channel %s...", config.ChannelID)
		if err := joinVoiceChannel(config.GuildID, config.ChannelID); err != nil {
			log.Printf("Failed to auto-connect: %v", err)
		}
	}

	// プログラムの終了を待機（Ctrl+Cで終了）
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("Bot is shutting down...")
	
	// 接続中なら切断
	if botState.voiceConnection != nil {
		leaveVoiceChannel()
	}
}

// messageCreate handles incoming messages
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if the bot is mentioned
	mentioned := false
	for _, user := range m.Mentions {
		if user.ID == s.State.User.ID {
			mentioned = true
			break
		}
	}

	if !mentioned {
		return
	}

	// Remove bot mention from message
	content := m.Content
	for _, user := range m.Mentions {
		if user.ID == s.State.User.ID {
			content = strings.Replace(content, "<@"+user.ID+">", "", -1)
			content = strings.Replace(content, "<@!"+user.ID+">", "", -1)
		}
	}
	content = strings.TrimSpace(content)

	// Parse command
	parts := strings.Fields(content)
	if len(parts) == 0 {
		s.ChannelMessageSend(m.ChannelID, "コマンドを指定してください！ `@Bot help` でヘルプを表示できます。")
		return
	}

	command := strings.ToLower(parts[0])

	switch command {
	case "join":
		handleJoinCommand(s, m, parts[1:])
	case "leave":
		handleLeaveCommand(s, m)
	case "status":
		handleStatusCommand(s, m)
	case "help":
		handleHelpCommand(s, m)
	default:
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("不明なコマンド: `%s`\n`@Bot help` でヘルプを表示できます。", command))
	}
}

// handleJoinCommand handles the join command
func handleJoinCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "チャンネル名またはメンションを指定してください！\n例: `@Bot join #雑談部屋`")
		return
	}

	guildID := m.GuildID
	var channelID string
	var channelName string

	// Check if it's a channel mention
	if strings.HasPrefix(args[0], "<#") && strings.HasSuffix(args[0], ">") {
		// Extract channel ID from mention
		channelID = strings.TrimPrefix(args[0], "<#")
		channelID = strings.TrimSuffix(channelID, ">")
	} else {
		// Search by channel name
		targetName := strings.Join(args, " ")
		targetName = strings.TrimPrefix(targetName, "#")

		// Get all channels in the guild
		channels, err := s.GuildChannels(guildID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("チャンネル一覧の取得に失敗しました: %v", err))
			return
		}

		// Find matching voice channel
		for _, ch := range channels {
			if ch.Type == discordgo.ChannelTypeGuildVoice && strings.EqualFold(ch.Name, targetName) {
				channelID = ch.ID
				channelName = ch.Name
				break
			}
		}

		if channelID == "" {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("ボイスチャンネル `%s` が見つかりませんでした。", targetName))
			return
		}
	}

	// Get channel info
	if channelName == "" {
		ch, err := s.Channel(channelID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("チャンネル情報の取得に失敗しました: %v", err))
			return
		}
		channelName = ch.Name
	}

	// Join voice channel
	if err := joinVoiceChannel(guildID, channelID); err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("ボイスチャンネルへの接続に失敗しました: %v", err))
		return
	}

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("✅ ボイスチャンネル `%s` に接続しました！", channelName))
}

// handleLeaveCommand handles the leave command
func handleLeaveCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	botState.RLock()
	connected := botState.voiceConnection != nil
	botState.RUnlock()

	if !connected {
		s.ChannelMessageSend(m.ChannelID, "現在、どのボイスチャンネルにも接続していません。")
		return
	}

	leaveVoiceChannel()
	s.ChannelMessageSend(m.ChannelID, "✅ ボイスチャンネルから退出しました。")
}

// handleStatusCommand handles the status command
func handleStatusCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	botState.RLock()
	defer botState.RUnlock()

	if botState.voiceConnection == nil {
		s.ChannelMessageSend(m.ChannelID, "📊 **Status**: ボイスチャンネルに接続していません")
		return
	}

	ch, err := s.Channel(botState.channelID)
	channelName := botState.channelID
	if err == nil {
		channelName = ch.Name
	}

	status := fmt.Sprintf("📊 **Status**\n"+
		"接続中: `%s`\n"+
		"ストリーミング: %v\n"+
		"オーディオデバイス: `%s`",
		channelName,
		botState.isStreaming,
		botState.audioDeviceName)

	s.ChannelMessageSend(m.ChannelID, status)
}

// handleHelpCommand handles the help command
func handleHelpCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	helpText := "**ConsoNance Bot - Commands**\n\n" +
		"`@Bot join #チャンネル名` - 指定したボイスチャンネルに接続します\n" +
		"`@Bot join チャンネル名` - チャンネル名で検索して接続します\n" +
		"`@Bot leave` - 現在のボイスチャンネルから退出します\n" +
		"`@Bot status` - 現在の接続状態を表示します\n" +
		"`@Bot help` - このヘルプを表示します"

	s.ChannelMessageSend(m.ChannelID, helpText)
}

// joinVoiceChannel joins a voice channel and starts streaming
func joinVoiceChannel(guildID, channelID string) error {
	botState.Lock()
	defer botState.Unlock()

	// If already connected, disconnect first
	if botState.voiceConnection != nil {
		log.Println("Already connected, disconnecting first...")
		botState.voiceConnection.Disconnect()
		if botState.isStreaming {
			botState.stopStreaming <- true
			botState.isStreaming = false
		}
	}

	// Join voice channel
	vc, err := session.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return fmt.Errorf("failed to join voice channel: %v", err)
	}

	botState.voiceConnection = vc
	botState.channelID = channelID
	botState.guildID = guildID

	// Wait for connection to be ready
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
			if vc.Ready {
				ready = true
				log.Println("Voice connection is ready!")
			}
		}
	}

	// Start streaming
	botState.isStreaming = true
	go func() {
		if err := streamSystemAudio(vc, botState.audioDeviceName); err != nil {
			log.Printf("Failed to stream system audio: %v", err)
			botState.Lock()
			botState.isStreaming = false
			botState.Unlock()
		}
	}()

	log.Printf("Successfully connected to voice channel: %s", channelID)
	return nil
}

// leaveVoiceChannel disconnects from the current voice channel
func leaveVoiceChannel() {
	botState.Lock()
	defer botState.Unlock()

	if botState.voiceConnection == nil {
		return
	}

	log.Println("Disconnecting from voice channel...")

	// Stop streaming
	if botState.isStreaming {
		botState.stopStreaming <- true
		botState.isStreaming = false
	}

	// Disconnect
	botState.voiceConnection.Disconnect()
	botState.voiceConnection = nil
	botState.channelID = ""

	log.Println("Disconnected from voice channel")
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

	log.Println("System audio streaming started!")

	// ストリーミング停止シグナルを待機
	<-botState.stopStreaming
	
	// デバイスの停止とクリーンアップ
	device.Stop()
	device.Uninit()
	
	log.Println("System audio streaming stopped.")
	return nil
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
