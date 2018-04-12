package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	pbm "github.com/cryptowatch/proto/markets"
	"github.com/cryptowatch/stream-client-go"
	"github.com/golang/protobuf/proto"
)

var (
	subs stringSlice

	detailed  = flag.Bool("detailed", false, "Print detailed contents of received messages")
	apiKey    = flag.String("apikey", "", "API key to use. Consider using -creds instead.")
	secretKey = flag.String("secretkey", "", "Secret key to use. Consider using -creds instead.")

	// TODO: -creds
)

func main() {
	flag.Var(&subs, "sub", "Subscription key. This flag can be given multiple times")
	flag.Parse()

	// Setup OS signal handler
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	args := flag.Args()

	// Get address to connect to
	u := ""
	if len(args) >= 1 {
		u = args[0]
	}

	// Setup market connection (but don't connect just yet)
	c, err := streamclient.NewStreamConn(&streamclient.StreamParams{
		URL: u,

		Reconnect:        true,
		ReconnectTimeout: 1 * time.Second,
		Backoff:          true,
		Subscriptions:    subs,

		APIKey:    *apiKey,
		SecretKey: *secretKey,
	})
	if err != nil {
		log.Fatal("%s", err)
	}

	// Will print state changes to the user
	c.AddStateListener(
		streamclient.StateAny,
		func(conn *streamclient.StreamConn, oldState, state streamclient.State, cause error) {
			fmt.Printf("State updated: %s -> %s", streamclient.StateNames[oldState], streamclient.StateNames[state])
			if cause != nil {
				fmt.Printf(" (%s)", cause)
			}
			fmt.Printf("\n")
		},
	)

	// Will print received market update messages
	c.AddMarketListener(func(conn *streamclient.StreamConn, msg *pbm.MarketUpdateMessage) {
		str := ""
		if *detailed {
			str = proto.MarshalTextString(msg)
		} else {
			str = proto.CompactTextString(msg)
		}

		fmt.Printf("Received market update: %s\n\n", str)
	})

	// Start connection loop
	fmt.Printf("Connecting to %s ...\n", c.URL())
	c.Connect()

	// Wait until the OS signal is received, at which point we'll close the
	// connection and quit
	<-interrupt
	fmt.Printf("Closing connection...\n")

	if err := c.Close(); err != nil {
		fmt.Printf("Failed to close connection: %s", err)
	}
}