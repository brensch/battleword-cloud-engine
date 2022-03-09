package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

const (
	// cloud run specific env vars
	// the vendor lock in is real
	EnvVarService       = "K_SERVICE"
	EnvVarRevision      = "K_REVISION"
	EnvVarConfiguration = "K_CONFIGURATION"
	EnvVarPort          = "PORT"

	defaultPort = "8080"

	FirestoreMatchCollection = "matches"
)

type apiStore struct {
	log      logrus.FieldLogger
	fsClient *firestore.Client
}

func main() {

	service := os.Getenv(EnvVarService)
	revision := os.Getenv(EnvVarRevision)
	configuration := os.Getenv(EnvVarConfiguration)
	// EnvVarService should always be set when running in a cloud run instance
	onCloud := service != ""

	ctx := context.Background()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	log.SetFormatter(&logrus.JSONFormatter{
		DisableTimestamp: false,
		TimestampFormat:  time.RFC3339Nano,
		// This is required for google logs to correctly assess log severity
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyLevel: "severity",
		},
	})

	// If we're not in a GCP environment, we need to give firebase credentials
	// otherwise the library automagically retrieves the service account from the environment
	var conf *firebase.Config
	var opts []option.ClientOption
	if !onCloud {
		conf = &firebase.Config{ProjectID: "battleword"}
		opts = append(opts, option.WithCredentialsFile("key.json"))
	}
	app, err := firebase.NewApp(ctx, conf, opts...)
	if err != nil {
		// Should not continue if firebase is not working
		log.WithError(err).Fatal("couldn't start firebase app")
	}

	fsClient, err := app.Firestore(ctx)
	if err != nil {
		// Should not continue if firestore is not working
		log.WithError(err).Fatal("couldn't start firebase app")
	}
	defer fsClient.Close()

	// Using env var since cloud run uses it
	port := os.Getenv(EnvVarPort)
	if port == "" {
		port = defaultPort
	}

	s := &apiStore{
		log:      log,
		fsClient: fsClient,
	}

	if onCloud {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(cors.Default())

	// Only run http logger locally - cloud has native api call logs we shouldn't duplicate
	if !onCloud {
		r.Use(MiddlewareLogger(log))
	}

	api := r.Group("/api")
	api.POST("/match", s.handleStartMatch)

	log.
		WithFields(logrus.Fields{
			"service":       service,
			"revision":      revision,
			"configuration": configuration,
		}).
		Info("app starting")

	r.Run(fmt.Sprintf(":%s", port))

}

func MiddlewareLogger(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Other handler can change c.Path so:
		path := c.Request.URL.Path
		start := time.Now()
		c.Next()
		latency := time.Since(start).Milliseconds()
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		clientUserAgent := c.Request.UserAgent()
		referer := c.Request.Referer()
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		dataLength := c.Writer.Size()
		if dataLength < 0 {
			dataLength = 0
		}

		entry := logrus.NewEntry(log).WithFields(logrus.Fields{
			"hostname":    hostname,
			"status_code": statusCode,
			"latency":     latency, // time to process
			"client_ip":   clientIP,
			"method":      c.Request.Method,
			"path":        path,
			"referer":     referer,
			"data_length": dataLength,
			"user_agent":  clientUserAgent,
		})

		if len(c.Errors) > 0 {
			entry.Error(c.Errors.ByType(gin.ErrorTypePrivate).String())
		} else {
			msg := "api call"
			if statusCode > 499 {
				entry.Error(msg)
			} else if statusCode > 399 {
				entry.Warn(msg)
			} else {
				entry.Info(msg)
			}
		}
	}
}
