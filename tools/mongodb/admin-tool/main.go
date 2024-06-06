package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/plgd-dev/device/v2/pkg/codec/json"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoArgs struct {
	TLS                   bool
	TLSCertificateKeyFile string
	TLSCAFile             string
	Eval                  string
	MongoURI              string
	Timeout               time.Duration
	DirectConnection      bool
}

func parseArgs() MongoArgs {
	var args MongoArgs

	flag.BoolVar(&args.TLS, "tls", false, "Enable TLS")
	flag.BoolVar(&args.TLS, "directConnection", false, "Direct connection")
	flag.StringVar(&args.TLSCertificateKeyFile, "tlsCertificateKeyFile", "", "Path to the TLS certificate key file")
	flag.StringVar(&args.TLSCAFile, "tlsCAFile", "", "Path to the TLS CA file")
	flag.StringVar(&args.Eval, "eval", "", "MongoDB eval command")
	flag.StringVar(&args.MongoURI, "uri", "mongodb://localhost:27017", "MongoDB URI")
	flag.DurationVar(&args.Timeout, "timeout", 10*time.Second, "Timeout for the eval command")

	flag.Parse()

	return args
}

func formatJSONKeys(jsonStr string) string {
	// Regex to find keys without double quotes and add double quotes around them
	re := regexp.MustCompile(`(?m)(\b\w+\b):`)
	formattedStr := re.ReplaceAllString(jsonStr, `"$1":`)
	return formattedStr
}

func parseEvalCommand(eval string) (interface{}, error) {
	re := regexp.MustCompile(`db\.adminCommand\((.*)\)`)
	matches := re.FindStringSubmatch(eval)
	if len(matches) < 2 {
		return nil, errors.New("invalid eval command format")
	}

	jsonString := matches[1]
	formattedJsonString := formatJSONKeys(jsonString)

	var command bson.D
	err := bson.UnmarshalExtJSON([]byte(formattedJsonString), false, &command)
	if err == nil {
		return command, nil
	}
	var stringCommand string
	err = bson.UnmarshalExtJSON([]byte(formattedJsonString), false, &stringCommand)
	if err == nil {
		formattedStringCommand := formatJSONKeys(stringCommand)
		err = bson.UnmarshalExtJSON([]byte(formattedStringCommand), false, &command)
		if err == nil {
			return command, nil
		}
		return map[string]interface{}{
			stringCommand: 1,
		}, nil
	}

	return nil, fmt.Errorf("failed to unmarshal eval command: %w", err)
}

func prepareClientOpts(args MongoArgs) (*options.ClientOptions, error) {
	clientOpts := options.Client().ApplyURI(args.MongoURI).SetDirect(args.DirectConnection)

	if !args.TLS {
		return clientOpts, nil
	}
	tlsConfig := &tls.Config{}
	if args.TLSCertificateKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(args.TLSCertificateKeyFile, args.TLSCertificateKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load key pair: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	if args.TLSCAFile != "" {
		caCert, err := os.ReadFile(args.TLSCAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}
		tlsConfig.RootCAs = x509.NewCertPool()
		if !tlsConfig.RootCAs.AppendCertsFromPEM(caCert) {
			return nil, errors.New("failed to append CA certificate")
		}
	}
	return clientOpts.SetTLSConfig(tlsConfig), nil
}

func run() error {
	args := parseArgs()

	// Setup MongoDB client options
	clientOpts, err := prepareClientOpts(args)
	if err != nil {
		return err
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer func() { _ = client.Disconnect(ctx) }()

	// Ping the database to verify the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	// Execute the eval command
	if args.Eval == "" {
		return nil
	}
	command, err := parseEvalCommand(args.Eval)
	if err != nil {
		return fmt.Errorf("failed to parse eval command: %w", err)
	}
	result := client.Database("admin").RunCommand(ctx, command)
	if result.Err() != nil {
		return result.Err()
	}

	// Output the result
	var res map[string]interface{}
	err = result.Decode(&res)
	if err != nil {
		return fmt.Errorf("failed to decode result: %w", err)
	}
	data, err := json.Encode(res)
	if err != nil {
		return fmt.Errorf("failed to encode result: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func main() {
	err := run()
	if err != nil {
		log.Fatalf("%v", err)
	}
}
