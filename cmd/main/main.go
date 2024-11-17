package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/sync/errgroup"

	"github.com/danielbukowski/twitch-chatbot/internal/access_credentials/cipher"
	"github.com/danielbukowski/twitch-chatbot/internal/access_credentials/storage"
	"github.com/danielbukowski/twitch-chatbot/internal/command"
	"github.com/danielbukowski/twitch-chatbot/internal/config"
	lg "github.com/danielbukowski/twitch-chatbot/internal/logger"
	"github.com/gempir/go-twitch-irc/v4"
	"github.com/nicklaw5/helix/v2"
	"go.uber.org/zap"
)

var meter = otel.Meter("main")
var chatMessageCounter metric.Int64Counter

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	isDevFlag := flag.Bool("dev", false, "development environment check")
	code := flag.String("code", "", "twitch authorization code to get access credentials")
	flag.Parse()

	cfg, err := config.New(*isDevFlag)
	if err != nil {
		panic(errors.Join(errors.New("failed to initialize config"), err))
	}

	logger, err := lg.New(*isDevFlag)
	if err != nil {
		panic(err)
	}

	shutdown, err := config.InitOpenTelemetrySDK(ctx, cfg.GrafanaCloudInstanceID, cfg.GrafanaAPIToken, *isDevFlag)
	if err != nil {
		panic(err)
	}

	accessCredentialsCipher, err := cipher.NewAESCipher(cfg.CipherPassphrase, 24)
	if err != nil {
		logger.Panic("failed to create AES cipher", zap.Error(err))
	}

	accessCredentialsStorage, err := storage.NewSQLiteStorage(ctx, "file:./db/database.db", cfg.DatabaseUsername, cfg.DatabasePassword, accessCredentialsCipher, logger)
	if err != nil {
		logger.Panic("failed to establish a connection to SQLite", zap.Error(err))
	}

	helixClient, err := helix.NewClient(&helix.Options{
		ClientID:     cfg.TwitchClientID,
		ClientSecret: cfg.TwitchClientSecret,
		RedirectURI:  cfg.TwitchOAuth2RedirectURI,
	})
	if err != nil {
		panic(err)
	}

	if *isDevFlag && len(*code) != 0 {
		logger.Info("exchanging authorization code for access credentials...")

		resp, err := helixClient.RequestUserAccessToken(*code)
		if err != nil || resp.StatusCode != 200 {
			logger.Panic("failed to exchange the code for access credentials", zap.Error(err))
		}

		err = accessCredentialsStorage.Save(ctx, resp.Data, cfg.TwitchChannelName)
		if err != nil {
			logger.Panic("failed to save the exchanged access credentials to database", zap.Error(err))
		}

		logger.Info("successfully exchanged and saved access credentials!")
	}

	accessCredentials, err := accessCredentialsStorage.Retrieve(ctx, cfg.TwitchChannelName)
	if err != nil {
		logger.Panic("failed to retrieve access credentials from the database", zap.Error(err))
	}

	ircClient := twitch.NewClient(cfg.TwitchChatbotName, fmt.Sprintf("oauth:%s", accessCredentials.AccessToken))
	ircClient.Join(cfg.TwitchChannelName)

	commandPrefix := "!"
	commandController := command.NewController(commandPrefix, logger)

	commandController.UseWith(command.ErrorHandler(logger))

	if *isDevFlag {

		// Add commands only after this line
		commandController.AddCommand(commandPrefix+"ping", command.Ping, []command.Filter{})
	}

	chatMessageCounter, err = meter.Int64Counter(
		"chat.message.counter",
		metric.WithDescription("Number of messages on the chat."),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		logger.Panic("failed to create a chat message counter", zap.Error(err))
	}

	ircClient.OnPrivateMessage(func(privateMessage twitch.PrivateMessage) {
		userMessage := privateMessage.Message

		if strings.EqualFold(privateMessage.User.Name, cfg.TwitchChatbotName) {
			return
		}

		if !strings.HasPrefix(userMessage, commandPrefix) {
			chatMessageCounter.Add(ctx, 1)
			return
		}

		commandController.CallCommand(ctx, userMessage, privateMessage, ircClient)
	})

	ircClient.OnConnect(func() {
		logger.Info("connected to the twitch chat!")
	})

	if isAccessTokenValid, _, err := helixClient.ValidateToken(accessCredentials.AccessToken); !isAccessTokenValid && err == nil {
		logger.Info("access credentials have expired")

		resp, err := helixClient.RefreshUserAccessToken(accessCredentials.RefreshToken)
		if err != nil || resp.StatusCode != 200 {
			logger.Panic("failed to refresh access credentials", zap.Error(err))
		}

		logger.Info("refreshed the expired access credentials")

		err = accessCredentialsStorage.Update(ctx, resp.Data, cfg.TwitchChannelName)
		if err != nil {
			logger.Panic("failed to update access credentials", zap.Error(err))
		}

		logger.Info("saved new access credentials to the database")

		ircClient.SetIRCToken(fmt.Sprintf("oauth:%s", resp.Data.AccessToken))
	}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		logger.Info("connecting to the twitch chat...")
		return ircClient.Connect()
	})

	g.Go(func() error {
		<-gCtx.Done()

		lg.Flush(logger)

		ctx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelShutdown()

		fmt.Println("closing the OpenTelemetry connections...")
		return shutdown(ctx)
	})

	g.Go(func() error {
		<-gCtx.Done()

		fmt.Println("closing the database connection...")
		return accessCredentialsStorage.Close()
	})

	g.Go(func() error {
		<-gCtx.Done()

		fmt.Println("closing the IRC server connection...")
		return ircClient.Disconnect()
	})

	if err = g.Wait(); err != nil {
		fmt.Printf("exited with error: %v\n", err)
		return
	}

	fmt.Println("gracefully exited without any errors!")
}
