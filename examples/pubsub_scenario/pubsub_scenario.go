package main

import (
	"context"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p-daemon/p2pclient"
	"github.com/libp2p/testlab/scenario"
	"github.com/sirupsen/logrus"
)

const topic = "load-test"

func subscribeReceivers(clients []*p2pclient.Client) {
	for _, client := range clients {
		client.Subscribe(context.Background(), topic)
		id, _, err := client.Identify()
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Infof("subscribed client %s", id.Pretty())
	}
}

func main() {
	logrus.SetLevel(logrus.InfoLevel)
	logrus.Info("Waiting 30s...")
	time.Sleep(time.Second * 30)
	logrus.Info("Starting")
	runner, err := scenario.NewScenarioRunner()
	if err != nil {
		logrus.Fatal(err)
	}

	peers, err := runner.Peers()
	if err != nil {
		logrus.Fatal(err)
	}

	if len(peers) < 2 {
		logrus.Fatalf("scenario needs at least 2 peers to run, found %d", len(peers))
	}

	sender := peers[0]
	receivers := peers[1:]
	go subscribeReceivers(receivers)

	for {
		wait := rand.Int63n(5000)
		logrus.Infof("Sending a message in %dms", wait)
		time.Sleep(time.Duration(wait) * time.Millisecond)
		data := make([]byte, rand.Intn(450)+50)
		rand.Read(data)
		err := sender.Publish(topic, data)
		if err != nil {
			logrus.Error(err)
		}
	}
}
