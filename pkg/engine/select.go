package engine

import (
	"git-indra.lan/indra-labs/indra/pkg/util/cryptorand"
)

// SelectUnusedCircuit accepts an array of 5 Node entries where all or some are
// empty and picks nodes for the remainder that do not have a hop at that
// position.
func (sm *SessionManager) SelectUnusedCircuit() (c [5]*Node) {
	sm.Lock()
	defer sm.Unlock()
	// Create a shuffled slice of Nodes to randomise the selection process.
	nodeList := make([]*Node, len(sm.nodes)-1)
	copy(nodeList, sm.nodes[1:])
	for i := range nodeList {
		if _, ok := sm.SessionCache[nodeList[i].ID]; !ok {
			log.T.F("adding session cache entry for node %s", nodeList[i].ID)
			sm.SessionCache[nodeList[i].ID] = &Circuit{}
		}
	}
	var counter int
out:
	for counter < 5 {
		for i := range sm.SessionCache {
			if counter == 5 {
				break out
			}
			if sm.SessionCache[i][counter] == nil {
				for j := range nodeList {
					if nodeList[j].ID == i {
						c[counter] = nodeList[j]
						counter++
						break
					}
				}
			}
		}
	}
	return
}

func (sm *SessionManager) SelectHops(hops []byte,
	alreadyHave Sessions) (so Sessions) {
	
	sm.Lock()
	defer sm.Unlock()
	ws := make(Sessions, 0)
out:
	for i := range sm.Sessions {
		if sm.Sessions[i] == nil {
			log.D.Ln("nil session", i)
			continue
		}
		for j := range alreadyHave {
			// We won't select any given in the alreadyHave list
			if alreadyHave[j] == nil {
				continue
			}
			if sm.Sessions[i].ID == alreadyHave[j].ID {
				continue out
			}
		}
		ws = append(ws, sm.Sessions[i])
	}
	// Shuffle the copy of the 
	cryptorand.Shuffle(len(ws), func(i, j int) {
		ws[i], ws[j] = ws[j], ws[i]
	})
	// Iterate the available sessions picking the first matching hop, then
	// prune it from the temporary slice and advance the cursor, wrapping
	// around at end.
	if len(alreadyHave) != len(hops) {
		log.E.Ln("selection requires equal length of hops and sessions")
		return
	}
	so = alreadyHave
	for i := range hops {
		var cur int
	out2:
		for {
			if ws[cur] == nil {
				cur++
				continue
			}
			if so[i] != nil {
				break out2
			}
			if ws[cur].Hop == hops[i] {
				so[i] = ws[cur]
				ws[cur] = nil
				cur = 0
				break
			}
			cur++
			if cur == len(ws) {
				cur = 0
			}
		}
	}
	log.D.Ln("selecting hops")
	return
}