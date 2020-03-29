package core

import (
	"time"

	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/bft"
)

/*
preprepare.go (part of core package) mainly implement functions on the preprepare step;
send /
*/

// sendPreprepare
func (c *core) sendPreprepare(request *bft.Request) {
	c.log.Debug("bft-1 sendPreprepare")
	// sequence is the proposal height and this node is the proposer
	// initiate the preprepare message and encode it
	c.log.Debug("[TEST sendPreprepare] current seq %d, proposal height %d, IsProposer %t", c.current.Sequence().Uint64(), request.Proposal.Height(), c.isProposer())
	if c.current.Sequence().Uint64() == request.Proposal.Height() && c.isProposer() {
		curView := c.currentView()
		preprepare, err := Encode(&bft.Preprepare{
			View:     curView,
			Proposal: request.Proposal,
		})
		if err != nil {
			c.log.Error("fail to encode preprepare state %d view %v", c.state, curView)
			return
		}
		// broadcast the message
		c.broadcast(&message{
			Code: msgPreprepare,
			Msg:  preprepare,
		})
		c.log.Info("sendPreprepare->broadcast->Post")

	}
}

//
// Decode -> checkMessage(make usre it is new) -> ensure it is from proposer -> verify proposal received -> accept preprepare
func (c *core) handlePreprepare(msg *message, src bft.Verifier) error {
	c.log.Debug("bft-1 handlePreprepare msg")
	c.log.Debug("from: ", src, "state:", c.state)
	// 1. Decode preprepare message first
	var preprepare *bft.Preprepare
	err := msg.Decode(&preprepare)
	if err != nil {
		c.log.Error("decode preprepare msg with err %s", err)
		return errDecodePreprepare
	}

	// we need to check the message: ensure we have the same view with the preprepare message
	// if not (namely, it is old message), see if we need to broadcast Commit.
	if err := c.checkMessage(msgPreprepare, preprepare.View); err != nil {
		c.log.Info("handle preprepare with err %s", err)
		if err == errOldMsg {
			// get all verifiers for this proposal
			verSet := c.server.ParentVerifiers(preprepare.Proposal).Copy()
			previousProposer := c.server.GetProposer(preprepare.Proposal.Height() - 1)
			verSet.CalcProposer(previousProposer, preprepare.View.Round.Uint64())
			// proposer matches (sequence + round) && given block exists
			// then broadcast commit
			if verSet.IsProposer(src.Address()) && c.server.HasPropsal(preprepare.Proposal.Hash()) {
				c.sendOldCommit(preprepare.View, preprepare.Proposal.Hash())
				return nil
			}
		}
		return err
	}

	// only proposer will broadcast preprepare message
	if !c.verSet.IsProposer(src.Address()) {
		c.log.Warn("igonore preprepare message since it is not the proposer")
		return errNotProposer
	}

	// verify the proposal we received
	if duration, err := c.server.Verify(preprepare.Proposal); err != nil {
		c.log.Warn("failed to verify proposal with err %s duration %d", err, duration)
		// it is a future block, and re-handle it after duration
		if err == consensus.ErrBlockCreateTimeOld {
			c.stopFuturePreprepareTimer() // stop timer
			c.futurePreprepareTimer = time.AfterFunc(duration, func() {
				c.sendEvent(backlogEvent{
					src: src,
					msg: msg,
				})
			})
		} else {
			c.sendNextRoundChange()
		}
		return err
	}
	c.log.Info("handlePreprepare->decodeMsg->checkMsg->Verify->AcceptPrepare->SetSate->sendCommit")

	// accept the preprepare message
	if c.state == StateAcceptRequest {
		if c.current.IsHashLocked() { // there is a locked proposal
			c.log.Debug("[TEST] hash is locked")
			if preprepare.Proposal.Hash() == c.current.GetLockedHash() { // at the same proposal
				c.acceptPreprepare(preprepare)
				c.setState(StatePrepared)
				c.sendCommit()
			} else { // at different proposals. change round
				c.sendNextRoundChange()
			}
		} else { // there is no locked proposal
			c.log.Debug("[TEST] hash is NOT locked")
			c.acceptPreprepare(preprepare)
			c.setState(StatePreprepared)
			c.sendPrepare()
		}
	}

	return nil
}

func (c *core) acceptPreprepare(preprepare *bft.Preprepare) {
	c.consensusTimestamp = time.Now()
	c.current.SetPreprepare(preprepare)
}
