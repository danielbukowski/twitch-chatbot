package main

import (
	"context"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/danielbukowski/twitch-chatbot/internal/access_credentials/cipher"
	"github.com/danielbukowski/twitch-chatbot/internal/access_credentials/storage"
	"github.com/danielbukowski/twitch-chatbot/internal/command"
	"github.com/danielbukowski/twitch-chatbot/internal/config"
	"github.com/danielbukowski/twitch-chatbot/internal/logger"
	"github.com/gempir/go-twitch-irc/v4"
	"github.com/nicklaw5/helix/v2"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()

	isDevEnv := flag.Bool("dev", false, "development environment check")
	code := flag.String("code", "", "twitch authorization code to get access credentials")
	flag.Parse()

	config, err := config.New(*isDevEnv)
	if err != nil {
		panic(errors.Join(errors.New("failed to initialize config"), err))
	}

	logger, err := logger.New(*isDevEnv)
	if err != nil {
		panic(err)
	}

	shutdown, err := initOpenTelemetrySDK(config.GrafanaCloudInstanceID, config.GrafanaAPIToken)
	if err != nil {
		panic(err)
	}

	defer func() {
		err := shutdown(ctx)
		if err != nil {
			fmt.Println(err)
		}
	}()

	//nolint:errcheck
	defer logger.Sync()

	logger.Info("successfully initialized logger", zap.Bool("IsDev", *isDevEnv))

	accessCredentialsCipher, err := cipher.NewAESCipher(config.CipherPassphrase, 24)
	if err != nil {
		logger.Panic("failed to create AES cipher", zap.Error(err))
	}

	accessCredentialsStorage, err := storage.NewSQLiteStorage(ctx, "file:./db/database.db", config.DatabaseUsername, config.DatabasePassword, accessCredentialsCipher, logger)
	if err != nil {
		logger.Panic("failed to establish a connection to SQLite", zap.Error(err))
	}

	helixClient, err := helix.NewClient(&helix.Options{
		ClientID:     config.TwitchClientID,
		ClientSecret: config.TwitchClientSecret,
		RedirectURI:  config.TwitchOAuth2RedirectURI,
	})
	if err != nil {
		panic(err)
	}

	if *isDevEnv && len(*code) != 0 {
		logger.Info("exchanging authorization code for access credentials...")

		resp, err := helixClient.RequestUserAccessToken(*code)
		if err != nil || resp.StatusCode != 200 {
			logger.Panic("failed to exchange the code for access credentials", zap.Error(err))
		}

		err = accessCredentialsStorage.Save(ctx, resp.Data, config.TwitchChannelName)
		if err != nil {
			logger.Panic("failed to save the exchanged access credentials to database", zap.Error(err))
		}

		logger.Info("successfully exchanged and saved access credentials!")
	}

	accessCredentials, err := accessCredentialsStorage.Retrieve(ctx, config.TwitchChannelName)
	if err != nil {
		logger.Panic("failed to retrieve access credentials from database", zap.Error(err))
	}

	ircClient := twitch.NewClient(config.TwitchChatbotName, fmt.Sprintf("oauth:%s", accessCredentials.AccessToken))
	ircClient.Join(config.TwitchChannelName)

	commandPrefix := "!"
	commandController := command.NewController(commandPrefix, logger)

	commandController.UseWith(command.ErrorHandler(logger))

	if *isDevEnv {

		// Add commands only after this line
		commandController.AddCommand(commandPrefix+"ping", command.Ping, []command.Filter{})
	}

	ircClient.OnPrivateMessage(func(privateMessage twitch.PrivateMessage) {
		userMessage := privateMessage.Message

		if !strings.HasPrefix(userMessage, commandPrefix) {
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

		logger.Info("refreshed access credentials")

		err = accessCredentialsStorage.Update(ctx, resp.Data, config.TwitchChannelName)
		if err != nil {
			logger.Panic("failed to update access credentials", zap.Error(err))
		}

		logger.Info("saved new access credentials to database")

		ircClient.SetIRCToken(fmt.Sprintf("oauth:%s", resp.Data.AccessToken))
	}

	err = ircClient.Connect()
	if err != nil {
		logger.Panic("failed to connect to the IRC server", zap.Error(err))
	}
}

func initOpenTelemetrySDK(instanceID, APIToken string) (shutdown func(context.Context) error, err error) {
	headers := make(map[string]string)
	credentials := base64.StdEncoding.EncodeToString([]byte(instanceID + ":" + APIToken))
	headers["Authorization"] = fmt.Sprintf("Basic %s", credentials)

	ctx := context.Background()
	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(propagator)

	metricExporter, err := otlpmetrichttp.New(
		ctx,
		otlpmetrichttp.WithHeaders(headers),
	)
	if err != nil {
		handleErr(err)
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter)),
		metric.WithReader(metric.NewManualReader(metric.WithProducer(runtime.NewProducer()))),
	)
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	err = runtime.Start()
	if err != nil {
		handleErr(err)
		return nil, err
	}

	err = host.Start(host.WithMeterProvider(meterProvider))
	if err != nil {
		handleErr(err)
		return nil, err
	}

	traceExporter, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithHeaders(headers),
	)
	if err != nil {
		handleErr(err)
		return nil, err
	}

	shutdownFuncs = append(shutdownFuncs, traceExporter.Shutdown)

	bsp := trace.NewBatchSpanProcessor(traceExporter)
	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithSpanProcessor(bsp),
	)
	shutdownFuncs = append(shutdownFuncs, traceProvider.Shutdown)
	otel.SetTracerProvider(traceProvider)

	logExporter, err := otlploghttp.New(ctx,
		otlploghttp.WithHeaders(headers),
	)
	if err != nil {
		handleErr(err)
		return nil, err
	}

	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
	)
	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	global.SetLoggerProvider(loggerProvider)

	return
}
