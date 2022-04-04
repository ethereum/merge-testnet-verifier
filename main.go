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
	"time"

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
	StopChan  chan interface{}
}

func (p *Verifier) StartProbes() {
	for _, vp := range p.Probes {
		vp := vp
		p.WaitGroup.Add(1)
		go func() {
			defer p.WaitGroup.Done()
			vp.Loop(p.StopChan)
		}()
	}
}

func (p *Verifier) WrapUp() bool {
	allSuccess := true
	for _, vp := range p.Probes {
		if vOut, err := vp.Verify(); err != nil {
			log15.Crit("Unable to perform verification", "client", vp.Client.ClientType(), "clientID", vp.Client.ClientID(), "verification", vp.Verification.VerificationName)
			allSuccess = false
		} else {
			var f func(string, ...interface{})
			if vOut.Success {
				f = log15.Info
			} else {
				f = log15.Crit
				allSuccess = false
			}
			f(vp.Verification.VerificationName, "client", vp.Client.ClientType(), "clientID", vp.Client.ClientID(), "pass", vOut.Success, "extra", vOut.Message)
		}
	}
	return allSuccess
}

func ContinuousCheckAllPassing(probes *VerificationProbes, stop <-chan interface{}, done chan<- interface{}) {
	for {
		select {
		case <-stop:
			return
		case <-time.After(time.Second):
			if probes.AllPassing() {
				close(done)
				return
			}
		}
	}
}

func TTDEpochTimeout(clients *Clients, maxEpochs uint64, stop <-chan interface{}, timeout chan<- interface{}) {
	beaconClients := clients.BeaconClients()
	<-time.After(time.Second * 60)
	for {
		select {
		case <-stop:
			return
		case <-time.After(time.Second):
			for _, bc := range beaconClients {
				if bc.TTDSlotNumber != nil {
					return
				}
				epoch, err := bc.GetOngoingEpochNumber()
				if err != nil {
					// Genesis has not occurred yet
					continue
				}
				if epoch >= maxEpochs {
					close(timeout)
					return
				}
			}
		}
	}
}

func VerificationsEpochTimeout(clients *Clients, maxEpochs uint64, stop <-chan interface{}, timeout chan<- interface{}) {
	beaconClients := clients.BeaconClients()
	for {
		select {
		case <-stop:
			return
		case <-time.After(time.Second):
			for _, bc := range beaconClients {
				if bc.TTDSlotNumber == nil {
					// TTD has not happened yet
					continue
				}
				ttdEpoch := bc.EpochForSlot(*bc.TTDSlotNumber)
				epoch, err := bc.GetOngoingEpochNumber()
				if err != nil {
					// Genesis has not occurred yet (this should be impossible ?)
					continue
				}
				if epoch >= ttdEpoch+maxEpochs+1 {
					close(timeout)
					return
				}
			}
		}
	}
}

func main() {
	var (
		clients             Clients
		ttdEpochLimit       uint64
		verifEpochLimit     uint64
		ttd                 TTD
		verifications       Verifications
		extra_verifications Verifications
	)
	flag.Var(&clients, "client",
		"Execution/Beacon client URL endpoint to check for the client's status in the form: <Client name>,http://<URL>:<IP>. Parameter can appear multiple times for multiple clients.")
	flag.Var(&ttd, "ttd", "Value of the Terminal Total Difficulty for the subscribed clients")
	flag.Var(&verifications, "override-verifications", "Path to verifications' YML file to override the defaults")
	flag.Var(&extra_verifications, "extra-verifications", "Path to verifications' YML file to append to the default verifications")
	flag.Uint64Var(&ttdEpochLimit, "ttd-epoch-limit", 5, "Max number of epochs to wait for the TTD to be reached. Disable timeout: 0. Default: 5")
	flag.Uint64Var(&verifEpochLimit, "verif-epoch-limit", 5, "Max number of epochs to wait for successful verifications after the merge has occurred. Disable timeout: 0. Default: 5")
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

	if len(extra_verifications) > 0 {
		verifications = append(verifications, extra_verifications...)
	}

	if len(verifications) == 0 {
		// We got no verifications
		log15.Crit("Zero verifications to perform.")
		os.Exit(1)
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
		log15.Crit("At least 1 execution layer verification is required (otherwise we cannot know when the terminal block has been found), exiting")
		os.Exit(1)
	}

	verifier.StopChan = make(chan interface{})

	verifier.StartProbes()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	allSuccess := make(chan interface{})
	ttdTimeout := make(chan interface{})
	verifTimeout := make(chan interface{})

	// Stop if all verifications have succeeded
	go ContinuousCheckAllPassing(&verifier.Probes, verifier.StopChan, allSuccess)

	// Stop if we reach a certain epoch from genesis and the TTD has not been reached yet
	if ttdEpochLimit > 0 {
		go TTDEpochTimeout(&clients, ttdEpochLimit, verifier.StopChan, ttdTimeout)
	}

	// Stop if we reach a certain epoch after the merge happened and the verifications have not passed
	if verifEpochLimit > 0 {
		go VerificationsEpochTimeout(&clients, verifEpochLimit, verifier.StopChan, verifTimeout)
	}

	select {
	case <-sigs:
		log15.Info("Received stop signal, wrapping up now")
	case <-allSuccess:
		log15.Info("All verifications have succeeded, wrapping up now")
	case <-ttdTimeout:
		log15.Info("Timeout while waiting for TTD to be reached, wrapping up now")
	case <-verifTimeout:
		log15.Info("Timeout while waiting for verifications to finish, wrapping up now")
	}
	// Need to wait here for the clients to finish up before continuing
	close(verifier.StopChan)
	if verifier.WrapUp() {
		// All verifications were successful
		os.Exit(0)
	}
	os.Exit(1)
}
