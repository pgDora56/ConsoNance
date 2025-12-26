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
	"bufio"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
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
	DiscordToken       string `yaml:"discord_token"`
	ChannelID          string `yaml:"channel_id"`
	GuildID            string `yaml:"guild_id"`
	AudioDeviceName    string `yaml:"audio_device_name"`
	AudioBufferPeriods int    `yaml:"audio_buffer_periods"` // 0 = use default (4)
}

// setupLogFile creates a log file and configures logging to both file and console
func setupLogFile() (*os.File, error) {
	// logsディレクトリを作成
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// タイムスタンプ付きのログファイル名を生成
	timestamp := time.Now().Format("20060102_150405")
	logFileName := filepath.Join(logsDir, fmt.Sprintf("consonance_%s.log", timestamp))

	// ログファイルを作成
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// コンソールとファイルの両方に出力するように設定
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	return logFile, nil
}

func main() {
	// バージョン情報を表示
	fmt.Println("==========================================")
	fmt.Printf("  %s\n", GetVersionString())
	fmt.Println("==========================================")
	fmt.Println()

	// ログファイルのセットアップ
	logFile, err := setupLogFile()
	if err != nil {
		log.Printf("Warning: Failed to setup log file: %v (continuing with console only)", err)
	} else {
		defer logFile.Close()
		log.Println("Log file created successfully")
	}

	// panicをキャッチしてログに記録
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC: %v", r)
			log.Println("Application terminated abnormally")
			waitForEnter()
		}
	}()

	// config.yamlの読み込み（存在しない場合は作成）
	config, err = loadOrCreateConfig()
	if err != nil {
		exitWithError("Failed to load config: %v", err)
	}

	// トークンの検証と対話的入力
	if config.DiscordToken == "" {
		token, err := promptForDiscordToken()
		if err != nil {
			exitWithError("Failed to get Discord token: %v", err)
		}
		config.DiscordToken = token
		log.Println("✓ Discord token saved to config.yaml")
	}
	// トークンの最初と最後の数文字だけ表示（セキュリティのため）
	tokenPreview := config.DiscordToken
	if len(tokenPreview) > 20 {
		tokenPreview = tokenPreview[:10] + "..." + tokenPreview[len(tokenPreview)-10:]
	}
	log.Printf("Using Discord token: %s", tokenPreview)

	// オーディオデバイスの選択
	selectedDevice := config.AudioDeviceName
	if selectedDevice == "" {
		// 設定ファイルに指定がない場合は、対話的に選択
		device, err := selectAudioDevice()
		if err != nil {
			exitWithError("Failed to select audio device: %v", err)
		}
		selectedDevice = device
		log.Printf("Selected audio device: %s", selectedDevice)
	} else {
		log.Printf("Using audio device from config: %s", selectedDevice)
	}

	// BotStateの初期化
	botState = &BotState{
		guildID:         config.GuildID,
		audioDeviceName: selectedDevice,
		stopStreaming:   make(chan bool),
	}

	// Discordセッションの作成
	session, err = discordgo.New("Bot " + config.DiscordToken)
	if err != nil {
		exitWithError("Failed to create Discord session: %v", err)
	}

	// Intentの設定
	session.Identify.Intents = discordgo.IntentsGuildVoiceStates | discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	// メッセージハンドラの登録
	session.AddHandler(messageCreate)

	// Discordセッションのオープン
	log.Println("Connecting to Discord...")
	if err := session.Open(); err != nil {
		log.Printf("Failed to open Discord session: %v", err)
		log.Println("")
		log.Println("=== Troubleshooting Authentication Error ===")
		log.Println("If you see '4004: Authentication failed', check the following:")
		log.Println("1. Verify your bot token is correct in config.yaml")
		log.Println("2. Go to Discord Developer Portal (https://discord.com/developers/applications)")
		log.Println("3. Select your application → Bot")
		log.Println("4. Under 'Privileged Gateway Intents', enable:")
		log.Println("   - MESSAGE CONTENT INTENT (required!)")
		log.Println("   - SERVER MEMBERS INTENT")
		log.Println("   - PRESENCE INTENT")
		log.Println("5. Save changes and try again")
		log.Println("6. If still failing, try resetting your bot token")
		log.Println("")
		waitForEnter()
		os.Exit(1)
	}
	defer session.Close()

	// Bot招待リンクを生成して表示
	if session.State.User != nil {
		clientID := session.State.User.ID
		// 必要な権限: Connect (1048576) + Speak (2097152) + View Channels (1024) + Send Messages (2048) + Read Message History (65536) = 3215376
		inviteURL := fmt.Sprintf("https://discord.com/api/oauth2/authorize?client_id=%s&scope=bot&permissions=3215376", clientID)
		fmt.Println("")
		fmt.Println("==========================================")
		fmt.Println("  Bot Invite Link:")
		fmt.Printf("  %s\n", inviteURL)
		fmt.Println("==========================================")
		fmt.Println("")
	}

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
	helpText := fmt.Sprintf("**%s - Commands**\n\n", GetVersionString()) +
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
	
	// 低遅延設定：バッファサイズを小さくする
	// frameSize (960 samples = 20ms) と同じサイズに設定
	deviceConfig.PeriodSizeInFrames = uint32(frameSize)
	
	// バッファの数を設定（デフォルト値: 4）
	bufferPeriods := config.AudioBufferPeriods
	if bufferPeriods == 0 {
		bufferPeriods = 4 // デフォルト値
	}
	deviceConfig.Periods = uint32(bufferPeriods)
	log.Printf("Audio buffer periods: %d (latency: ~%dms)", bufferPeriods, bufferPeriods*20)

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

// selectAudioDevice displays available audio devices and lets the user select one
func selectAudioDevice() (string, error) {
	// malgoコンテキストの初期化
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return "", fmt.Errorf("failed to initialize malgo context: %v", err)
	}
	defer func() {
		_ = ctx.Uninit()
		ctx.Free()
	}()

	// 再生デバイス（ループバックに使用可能）の取得
	infos, err := ctx.Devices(malgo.Playback)
	if err != nil {
		return "", fmt.Errorf("failed to get playback devices: %v", err)
	}

	if len(infos) == 0 {
		return "", fmt.Errorf("no playback devices found")
	}

	// デバイス一覧を表示
	fmt.Println("\n=== Select Audio Device ===")
	fmt.Println("Available audio devices for loopback capture:")
	fmt.Println()

	for i, info := range infos {
		defaultMark := ""
		if info.IsDefault > 0 {
			defaultMark = " (Default)"
		}
		fmt.Printf("[%d] %s%s\n", i+1, info.Name(), defaultMark)
	}

	fmt.Println()
	fmt.Print("Enter device number: ")

	// ユーザー入力を読み取る
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %v", err)
	}

	// 入力をトリムして数値に変換
	input = strings.TrimSpace(input)
	selection, err := strconv.Atoi(input)
	if err != nil {
		return "", fmt.Errorf("invalid input: please enter a number")
	}

	// 選択範囲チェック
	if selection < 1 || selection > len(infos) {
		return "", fmt.Errorf("invalid selection: please enter a number between 1 and %d", len(infos))
	}

	// 選択されたデバイスの名前を返す
	selectedDevice := infos[selection-1].Name()
	fmt.Printf("\n✓ Selected: %s\n", selectedDevice)

	// デフォルトとして保存するか確認
	fmt.Print("\nSave this device as default in config.yaml? (y/n): ")
	saveInput, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Warning: Failed to read input: %v", err)
		fmt.Println()
		return selectedDevice, nil
	}

	saveInput = strings.TrimSpace(strings.ToLower(saveInput))
	if saveInput == "y" || saveInput == "yes" {
		if err := saveDeviceToConfig(selectedDevice); err != nil {
			log.Printf("Warning: Failed to save device to config: %v", err)
			fmt.Println("Device selection will be used for this session only.")
		} else {
			fmt.Println("✓ Device saved to config.yaml")
		}
	}

	fmt.Println()
	return selectedDevice, nil
}

// saveDeviceToConfig saves the selected audio device to config.yaml
func saveDeviceToConfig(deviceName string) error {
	// config.yamlを読み込む
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return fmt.Errorf("failed to read config.yaml: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	updated := false
	inAudioSection := false

	// 既存の audio_device_name を探して更新、またはコメントアウトされている行を置き換え
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// オーディオデバイスセクションを検出
		if strings.Contains(trimmed, "オーディオデバイスの設定") || 
		   strings.Contains(trimmed, "Audio Device Settings") {
			inAudioSection = true
			continue
		}

		// 別のセクションに入ったらフラグをオフ
		if inAudioSection && strings.HasPrefix(trimmed, "#") && 
		   (strings.Contains(trimmed, "デバイス一覧") || strings.Contains(trimmed, "Device List")) {
			inAudioSection = false
		}

		// audio_device_name の行を見つけた場合
		if strings.HasPrefix(trimmed, "audio_device_name:") || 
		   strings.HasPrefix(trimmed, "# audio_device_name:") {
			lines[i] = fmt.Sprintf("audio_device_name: \"%s\"", deviceName)
			updated = true
			break
		}
	}

	// audio_device_name が見つからなかった場合は、ファイルの最後に追加
	if !updated {
		lines = append(lines, fmt.Sprintf("audio_device_name: \"%s\"", deviceName))
	}

	// ファイルに書き込む
	output := strings.Join(lines, "\n")
	if err := os.WriteFile("config.yaml", []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write config.yaml: %v", err)
	}

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

// waitForEnter waits for the user to press Enter before exiting
func waitForEnter() {
	fmt.Println("")
	fmt.Print("Press Enter to exit...")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
}

// exitWithError logs an error message and waits for Enter before exiting
func exitWithError(format string, args ...interface{}) {
	log.Printf(format, args...)
	waitForEnter()
	os.Exit(1)
}

// loadOrCreateConfig loads config.yaml or creates it if it doesn't exist
func loadOrCreateConfig() (*Config, error) {
	// config.yamlが存在するか確認
	if _, err := os.Stat("config.yaml"); os.IsNotExist(err) {
		// 存在しない場合は、テンプレートから作成
		log.Println("config.yaml not found. Creating a new one...")
		if err := createDefaultConfig(); err != nil {
			return nil, fmt.Errorf("failed to create config.yaml: %v", err)
		}
		log.Println("✓ Created config.yaml")
	}

	// config.yamlを読み込む
	configFile, err := os.Open("config.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to open config.yaml: %v", err)
	}
	defer configFile.Close()

	cfg := &Config{}
	decoder := yaml.NewDecoder(configFile)
	if err := decoder.Decode(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config.yaml: %v", err)
	}

	return cfg, nil
}

// createDefaultConfig creates a default config.yaml file
func createDefaultConfig() error {
	defaultConfig := `# ConsoNance Configuration File
# This file was automatically generated

# Discord Bot Token (REQUIRED)
# Get your token from https://discord.com/developers/applications
discord_token: ""

# ===== Optional Settings =====
# These settings are all optional. You can control the bot via Discord chat commands!

# Auto-connect Settings (Optional)
# If you want the bot to automatically join a voice channel on startup, 
# uncomment and fill in these values:
# guild_id: "YOUR_GUILD_ID_HERE"
# channel_id: "YOUR_CHANNEL_ID_HERE"

# To join via Discord chat, simply mention the bot:
#   @Bot join #channel-name
#   @Bot leave
#   @Bot status
#   @Bot help

# Audio Device Settings (Optional)
# If not specified, you'll be prompted to select a device from a list at startup
# To use a specific device, uncomment and set the device name:
# audio_device_name: "Speakers (Realtek High Definition Audio)"

# Audio Buffer Settings (Optional)
# Number of audio buffer periods (affects latency and stability)
# 0 = use default (4), higher values = more stable but more latency
# Recommended: 3-6 depending on your system performance
audio_buffer_periods: 0
`

	if err := os.WriteFile("config.yaml", []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config.yaml: %v", err)
	}

	return nil
}

// promptForDiscordToken prompts the user to enter their Discord bot token
func promptForDiscordToken() (string, error) {
	fmt.Println("\n=== Discord Bot Token Required ===")
	fmt.Println("Your Discord bot token is not configured.")
	fmt.Println()
	fmt.Println("To get your bot token:")
	fmt.Println("1. Go to https://discord.com/developers/applications")
	fmt.Println("2. Select your application (or create a new one)")
	fmt.Println("3. Navigate to the 'Bot' section")
	fmt.Println("4. Click 'Reset Token' or 'Copy' to get your token")
	fmt.Println("5. Make sure to enable 'MESSAGE CONTENT INTENT' under Privileged Gateway Intents")
	fmt.Println()
	fmt.Print("Paste your Discord bot token here: ")

	reader := bufio.NewReader(os.Stdin)
	token, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read token: %v", err)
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("token cannot be empty")
	}

	// トークンをconfig.yamlに保存
	if err := saveTokenToConfig(token); err != nil {
		return "", fmt.Errorf("failed to save token to config: %v", err)
	}

	return token, nil
}

// saveTokenToConfig saves the Discord token to config.yaml
func saveTokenToConfig(token string) error {
	// config.yamlを読み込む
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return fmt.Errorf("failed to read config.yaml: %v", err)
	}

	lines := strings.Split(string(data), "\n")

	// discord_token の行を探して更新
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "discord_token:") {
			lines[i] = fmt.Sprintf("discord_token: \"%s\"", token)
			break
		}
	}

	// ファイルに書き込む
	output := strings.Join(lines, "\n")
	if err := os.WriteFile("config.yaml", []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write config.yaml: %v", err)
	}

	return nil
}
