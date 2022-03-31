package main

import (
	"flag"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"gopkg.in/inconshreveable/log15.v2"
)

type TTD struct {
	*big.Int
}

func (t *TTD) Set(val string) error {
	dec := big.NewInt(0)
	if _, suc := dec.SetString(val, 0); !suc {
		return fmt.Errorf("unable to parse %s", val)
	}
	*t = TTD{dec}
	return nil
}

type Verifier struct {
	Clients   Clients
	Probes    VerificationProbes
	WaitGroup sync.WaitGroup
	StopChan  chan struct{}
}

func (p *Verifier) StartProbes() {
	p.StopChan = make(chan struct{})
	for _, vp := range p.Probes {
		vp := vp
		p.WaitGroup.Add(1)
		go func() {
			defer p.WaitGroup.Done()
			vp.Loop(p.StopChan)
		}()
	}
}

func (p *Verifier) StopProbes() {
	close(p.StopChan)
}

func (p *Verifier) WrapUp() {
	for _, vp := range p.Probes {
		if vOut, err := vp.Verify(); err != nil {
			log15.Crit("Unable to perform verification", "client", vp.Client.ClientType(), "clientID", vp.Client.ClientID(), "verification", vp.Verification.VerificationName)
		} else {
			var f func(string, ...interface{})
			if vOut.Success {
				f = log15.Info
			} else {
				f = log15.Crit
			}
			f(vp.Verification.VerificationName, "client", vp.Client.ClientType(), "clientID", vp.Client.ClientID(), "pass", vOut.Success, "extra", vOut.Message)
		}
	}
}

func main() {
	var (
		clients       Clients
		ttd           TTD
		verifications Verifications
	)
	flag.Var(&clients, "client",
		"Execution/Beacon client URL endpoint to check for the client's status in the form: <Client name>,http://<URL>:<IP>")
	flag.Var(&ttd, "ttd",
		"Value of the Terminal Total Difficulty for the subscribed clients")
	flag.Var(&verifications, "verifications",
		"Value of the Terminal Total Difficulty for the subscribed clients")
	flag.Parse()

	verifier := Verifier{
		Clients: clients,
		Probes:  make(VerificationProbes, 0),
	}

	updateAllTTDTimestamps := func(timestamp uint64) {
		for _, cl := range clients {
			if cl.ClientLayer() == Beacon {
				bc := cl.(*BeaconClient)
				bc.UpdateTTDTimestamp(timestamp)
			}
		}
	}

	if len(verifications) == 0 {
		// Try to use default_verifications.yml
		if path, err := os.Getwd(); err == nil {
			defaultVerificationsPath := filepath.Join(path, "default_verifications.yml")
			if _, err = os.Stat(defaultVerificationsPath); err == nil {
				verifications.Set(defaultVerificationsPath)
			}
		}
	}

	for _, cl := range clients {
		if cl.ClientLayer() == Beacon {
			bc := cl.(*BeaconClient)
			bc.TTD = ttd
		} else if cl.ClientLayer() == Execution {
			el := cl.(*ExecutionClient)
			el.TTD = ttd
			el.UpdateTTDTimestamp = updateAllTTDTimestamps
		}
		clientProbes := NewVerificationProbes(cl, verifications)
		for _, cp := range clientProbes {
			cp.AllProbesClient = &clientProbes
		}
		verifier.Probes = append(verifier.Probes, clientProbes...)
	}

	if verifier.Probes.ExecutionVerifications() == 0 {
		log15.Crit("At least 1 execution layer verification is required, exiting")
		os.Exit(1)
	}

	verifier.StartProbes()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs
	// Need to wait here for the clients to finish up before continuing
	verifier.StopProbes()
	verifier.WrapUp()
}
