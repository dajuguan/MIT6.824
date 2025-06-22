package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"
	"bytes"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.824/labgob"
	"6.824/labgob"
	"6.824/labrpc"
)

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

const (
	HeartBeatInterval = 100 * time.Millisecond
)

type Role int

const (
	Follower Role = iota + 1
	Candicate
	Leader
)

func (r Role) String() string {
	switch r {
	case Follower:
		return "Follower"
	case Candicate:
		return "Candicate"
	case Leader:
		return "Leader"
	default:
		return "Unknown"
	}
}

type Log struct {
	Term  int
	Index int
	Cmd   interface{}
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	CurrentTerm int
	VotedFor    int // We should only reset votedFor when receiving a RPC req/resp with a higher term, or we might vote more than once in the same term
	role        Role
	lastLive    time.Time
	// for 2B
	Log         []Log
	commitIndex int
	applyCh     chan ApplyMsg
	nextIndex   []int
	matchIndex  []int
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (2A).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	term = rf.CurrentTerm
	isleader = rf.role == Leader
	return term, isleader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
func (rf *Raft) persist() {
	// Your code here (2C).
	// Be cautious to use defer rf.persis() in other methods, cause the applyCh is blocking, we might haven't persisted the log, then suddenly the server is killed
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.CurrentTerm)
	e.Encode(rf.VotedFor)
	e.Encode(rf.Log)
	data := w.Bytes()
	rf.persister.SaveRaftState(data)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var currentTerm int
	var votedFor int
	var log []Log
	if d.Decode(&currentTerm) != nil || d.Decode(&votedFor) != nil || d.Decode(&log) != nil {
		panic("readPersist decode error")
	} else {
		rf.CurrentTerm = currentTerm
		rf.VotedFor = votedFor
		rf.Log = log
	}
}

// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {

	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

type RequestAppendEntry struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []Log
	LeaderCommit int
}

func (r RequestAppendEntry) String() string {
	return fmt.Sprintf("term %v, leaderId:%v, prevLogIndex:%v, prevLogTerm: %v, commit: %v, entries: %v", r.Term, r.LeaderId, r.PrevLogIndex, r.PrevLogTerm, r.LeaderCommit, len(r.Entries))
}

type AppendEntriesReply struct {
	Term    int
	Success bool
	// Quick append log
	Xterm int // the conflicting term
	// the conflicting index
	// If Xterm = -2, it means timeout, we just return and don't deal with it
	// if it's Xterm is -1, XIndex is the nextIndex to start append;
	// else: XIndex is the conflicting index
	Xindex int // If XIndex = -1, leader's request is rejected, nothing to do; else set nextIndex according to XIndex and Xterm
}

func (rf *Raft) sendAppendEntries(server int, args *RequestAppendEntry, reply *AppendEntriesReply) {
	ok := rf.peers[server].Call("Raft.RequestAppendEntries", args, reply)
	// DPrintf("leader %v, RequestAppendEntries reply from server: %v, success: %v", rf.me, server, reply.Success)

	rf.mu.Lock()
	defer rf.mu.Unlock()
	if ok && reply.Term > rf.CurrentTerm {
		rf.CurrentTerm = reply.Term
		rf.role = Follower
		rf.VotedFor = -1

		rf.persist()
	}

	if rf.role != Leader || !ok {
		return
	}

	if reply.Success {
		// update nextIndex and matchIndex
		rf.nextIndex[server] = args.PrevLogIndex + 1 + len(args.Entries)
		rf.matchIndex[server] = rf.nextIndex[server] - 1
		DPrintf("reply success, from peer: %v, prevLogIndex: %v, newLogIndex: %v", server, args.PrevLogIndex, rf.nextIndex[server])
		DPrintf("check commitIndex, leader: %v, matchIndex: %v", rf.me, rf.matchIndex)

		// update commitIndex
		for i := len(rf.Log) - 1; i > rf.commitIndex; i-- {
			if rf.Log[i].Term == rf.CurrentTerm {
				count := 0
				for j := range rf.peers {
					if j == rf.me || rf.matchIndex[j] >= i {
						count += 1
					}
				}
				if count > len(rf.peers)/2 {
					// send ApplyMsg to applyCh for tester
					i := i

					for commitedInd := rf.commitIndex + 1; commitedInd <= i; commitedInd++ {
						cmd := rf.Log[commitedInd].Cmd
						commitedInd := applyChIndex(commitedInd)
						// must block to avoid out of order apply
						rf.applyCh <- ApplyMsg{
							CommandValid: true,
							Command:      cmd,
							CommandIndex: commitedInd,
						}
					}

					rf.commitIndex = i
					DPrintf("leader %v, set commit index: %v", rf.me, applyChIndex(i))
					// Allservers: update lastApplied and apply log[lastApplied] to state machine
					// todo
					break
				}
			}
		}
	} else if reply.Xterm != -2 {
		// Although, ok is true and reply.Success is false, we still have to check Xterm!=-2, cause RPC might early return our response before the reply is fully set in requestAppendEntries.
		// If AppendEntries fails because of log inconsistency, do quick recovery.
		// Must be ok, or it'll reset nextIndex if timeout due to the target peer is disconnected.
		if reply.Xterm == -1 {
			// skip to follower's index
			rf.nextIndex[server] = reply.Xindex
		} else {
			// skip to leader's last index of previous term
			for i := args.PrevLogIndex; i >= -1; i-- {
				if i == -1 || rf.Log[i].Term != args.PrevLogTerm {
					rf.nextIndex[server] = i + 1
					break
				}
			}
		}
	}
	// else, timeout shoule be retried in sendHeartBeats, don't retry here cause leader might has changed
}

func applyChIndex(index int) int {
	// ApplyMsg.CommandIndex starts from 0, while here CommandIndex is initialized with -1, so we need to add 1
	return index + 1
}

func (rf *Raft) RequestAppendEntries(args *RequestAppendEntry, reply *AppendEntriesReply) {
	DPrintf("leader: %v, RequestAppendEntries,prevIndex:%v,prevTerm:%v, prevCommit:%v,leaderTerm:%v,entries:%v, to peer: %v under mycommitIndex: %v, myterm: %v", args.LeaderId, args.PrevLogIndex, args.PrevLogTerm, args.LeaderCommit, args.Term, len(args.Entries), rf.me, rf.commitIndex, rf.CurrentTerm)
	// election timeouts, reset timer
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Success = false
	reply.Term = rf.CurrentTerm
	reply.Xindex = -1
	reply.Xterm = -2

	// cond:1
	if args.Term < rf.CurrentTerm {
		return
	}

	// Allservers: transition to follower if term >= currentTerm
	rf.CurrentTerm = args.Term
	rf.role = Follower

	rf.lastLive = time.Now()
	// ensure only vote once for each term
	if args.Term > rf.CurrentTerm {
		rf.VotedFor = -1
	}

	// cond:2
	if args.PrevLogIndex > -1 {
		// check if prevLogIndex is in the log
		if args.PrevLogIndex > len(rf.Log)-1 {
			reply.Xterm = -1
			reply.Xindex = len(rf.Log)

			rf.persist()
			return
		}
		if rf.Log[args.PrevLogIndex].Term != args.PrevLogTerm {
			// log mismatch, return false
			reply.Xterm = rf.Log[args.PrevLogIndex].Term
			reply.Xindex = args.PrevLogIndex

			rf.persist()
			return
		}
	}

	// cond:3&4
	for i, entry := range args.Entries {
		logIndex := args.PrevLogIndex + 1 + i
		var existing *Log
		if len(rf.Log) > 0 && logIndex >= 0 && len(rf.Log) > logIndex {
			existing = &rf.Log[logIndex]
		}
		if existing != nil {
			if existing.Term != entry.Term { // same index but different term or different command
				// cond:3 Delete the existing entry and all that follow it
				rf.Log = rf.Log[:logIndex]
				rf.Log = append(rf.Log, entry)
			}
			continue
		} else { // existing must be nil to avoid applying the same entry twice, which happens when leader sends heartbeat
			// cond:4 Append any new entries not already in the log
			rf.Log = append(rf.Log, entry)
		}
	}
	rf.persist() // log changes in the following

	// cond:5
	if args.LeaderCommit > rf.commitIndex {
		// the right one shouldn't be len(rf.log) -1, because the old leader might has append new logs longer, so old leader (current follower) log lenght might be longer than new leader's log length. If we use len(rf.log)-1, then follower might commit some logs not belong to the new leader
		commitIndex := min(args.LeaderCommit, args.PrevLogIndex+len(args.Entries))
		// Send ApplyMsg for each new committed log entry for tester
		for i := rf.commitIndex + 1; i <= commitIndex; i++ {
			cmd := rf.Log[i].Cmd
			commitedInd := applyChIndex(i)
			// must block to avoid out of order apply
			rf.applyCh <- ApplyMsg{
				CommandValid: true,
				Command:      cmd,
				CommandIndex: commitedInd,
			}
			DPrintf("args:%v, follower:%v,  applyCmd: %v, cmdIndex: %v", args, rf.me, cmd, commitedInd)
		}
		DPrintf("follower:%v,  set commit index: %v", rf.me, applyChIndex(commitIndex))
		rf.commitIndex = commitIndex
	}
	// Allservers: update lastApplied and apply log[lastApplied] to state machine
	// todo
	DPrintf("leader: %v, RequestAppendEntries success,prevIndex:%v,prevTerm:%v, prevCommit:%v,leaderTerm:%v,entries:%v, to peer: %v under mycommitIndex: %v, myterm: %v, myLog: %v", args.LeaderId, args.PrevLogIndex, args.PrevLogTerm, args.LeaderCommit, args.Term, len(args.Entries), rf.me, rf.commitIndex, rf.CurrentTerm, len(rf.Log))
	reply.Success = true
}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateId  int
	LastLogTerm  int
	LastLogIndex int
}

func (r RequestVoteArgs) String() string {
	return fmt.Sprintf("term:%v,id:%v,lastLogTerm:%v,lastLogIndex:%v", r.Term, r.CandidateId, r.LastLogTerm, r.LastLogIndex)
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	lastTerm, lastIndex := rf.LastLog()
	DPrintf("server %v, get requestVote: %v, curTerm:%v, lastTerm:%v, lastIndex:%v,role:%v,votedFor:%v", rf.me, args, rf.CurrentTerm, lastTerm, lastIndex, rf.role, rf.VotedFor)
	reply.VoteGranted = false

	reply.Term = rf.CurrentTerm
	if args.Term < rf.CurrentTerm {
		return
	}

	defer rf.persist()

	// transition to follower,  reset currentTerm and votedFor. 缺一不可!
	// currentTerm must be updated to ensure one peer is only allowed to vote once for each term
	if args.Term > rf.CurrentTerm {
		rf.role = Follower
		// make sure kickoff next election use latest term
		rf.CurrentTerm = args.Term
		rf.VotedFor = -1
	}

	// lastLive should only be updated after granting vote, or the current server will issue leader election again
	if rf.VotedFor == -1 || rf.VotedFor == args.CandidateId {
		// candidate's log must be at least update to date with the majority.
		// shouldn't compare rf.currentTerm!

		lastTerm, lastIndex := rf.LastLog()
		if args.LastLogTerm > lastTerm {
			reply.VoteGranted = true
			rf.VotedFor = args.CandidateId
			rf.lastLive = time.Now()
			return
		}
		if args.LastLogTerm == lastTerm {
			if args.LastLogIndex >= lastIndex {
				reply.VoteGranted = true
				rf.VotedFor = args.CandidateId
				rf.lastLive = time.Now()
			}
		}
	}
}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	DPrintf("send requestVote from: %v, to :%v, term: %v", rf.me, server, rf.CurrentTerm)
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := false

	// Your code here (2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	term = rf.CurrentTerm
	index = len(rf.Log)
	isLeader = rf.role == Leader

	if !isLeader {
		return rf.commitIndex, term, isLeader
	}

	defer rf.persist()

	newLog := Log{
		Term:  rf.CurrentTerm,
		Cmd:   command,
		Index: index,
	}

	DPrintf("leader:%v append log:%v", rf.me, newLog)
	// update log
	rf.Log = append(rf.Log, newLog)

	// send new entries to other peers immdiately so that even if leader request appendEntries rpc timeout, follower will have a chance to receive the new cmd
	rf._sendHeartBeats()
	return applyChIndex(index), term, isLeader
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) sendHeartBeats() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf._sendHeartBeats()
}

// not threadsafe, must use it under lock
func (rf *Raft) _sendHeartBeats() {
	if rf.role != Leader {
		return
	}
	rf.lastLive = time.Now()

	// send heartbeat
	emptyEntry := RequestAppendEntry{
		Term:         rf.CurrentTerm,
		LeaderId:     rf.me,
		LeaderCommit: rf.commitIndex,
	}

	for server := range rf.peers {
		if server == rf.me {
			continue
		}
		// use goroutine for each rpc to avoid blocking
		server := server
		var entries []Log
		nextIndex := rf.nextIndex[server]
		if len(rf.Log)-1 >= nextIndex && nextIndex >= 0 {
			entries = rf.Log[nextIndex:]
		}
		prevLogIndex := nextIndex - 1
		prevLogTerm := -1
		if prevLogIndex >= 0 {
			prevLogTerm = rf.Log[prevLogIndex].Term
		}
		go func() {
			reply := AppendEntriesReply{}
			entry := emptyEntry
			entry.Entries = entries
			entry.PrevLogIndex = prevLogIndex
			entry.PrevLogTerm = prevLogTerm

			rf.sendAppendEntries(server, &entry, &reply)
		}()
	}
}

// Mustn't use seed or it'll generate the same random number every time
func RandElectionTimeout() time.Duration {
	randVal := rand.Float32()*2.5 + 2.5 // use a larger timeouts, cause timeout is more frequent in lab2C
	duration := int64(float32(HeartBeatInterval) * randVal)
	return time.Duration(duration)
}

func (rf *Raft) LastLog() (int, int) {
	lastTerm := -1
	lastIndex := -1
	if len(rf.Log) > 0 {
		lastTerm = rf.Log[len(rf.Log)-1].Term
		lastIndex = len(rf.Log) - 1
	}
	return lastTerm, lastIndex
}

func (rf *Raft) kickOffLeaderElection(electionTimeout time.Duration) {
	rf.mu.Lock()
	rf.role = Candicate
	rf.CurrentTerm += 1
	rf.VotedFor = rf.me
	rf.lastLive = time.Now()
	term := rf.CurrentTerm

	rf.persist()

	rf.mu.Unlock()
	start := time.Now()

	go func() {
		// send Rpc request to other servers
		var mu sync.Mutex
		// cond share the same lock with mu cause they control the same shared variables
		cond := sync.NewCond(&mu)
		totalVotes := 1
		finished := 1

		lastTerm, lastIndex := rf.LastLog()
		requestVotes := RequestVoteArgs{
			Term:         term,
			CandidateId:  rf.me,
			LastLogTerm:  lastTerm,
			LastLogIndex: lastIndex,
		}

		for server := range rf.peers {
			if server == rf.me {
				continue
			}
			go func(server int) {
				var reply RequestVoteReply
				ok := rf.sendRequestVote(server, &requestVotes, &reply)
				mu.Lock()
				defer mu.Unlock()
				if ok && reply.VoteGranted {
					DPrintf("server %v got vote from %v, reply: %v", rf.me, server, reply)
					totalVotes += 1
				}
				finished += 1
				cond.Broadcast()
			}(server)
		}

		cond.L.Lock()
		for totalVotes < (len(rf.peers)/2+1) && finished != len(rf.peers) {
			DPrintf("candidate: %v, total votes: %v, finished: %v", rf.me, totalVotes, finished)
			cond.Wait()
		}

		if totalVotes > len(rf.peers)/2 {
			rf.mu.Lock()
			defer rf.mu.Unlock()
			if rf.role == Candicate && rf.CurrentTerm == term && time.Since(start) < electionTimeout {
				DPrintf("peer: %v, selected as leader", rf.me)
				rf.role = Leader
				// rf.VotedFor = -1 shouldn't reset votedFor here, cause we might vote for other candidates in the same term
				rf.nextIndex = make([]int, len(rf.peers))
				rf.matchIndex = make([]int, len(rf.peers))

				for i := range rf.nextIndex {
					// initialize to leader last log index + 1
					rf.nextIndex[i] = len(rf.Log)
				}
				for i := range rf.nextIndex {
					// initialize to -1
					rf.matchIndex[i] = -1
				}

				// commit a new empty log entry to avoid liveness issue, when the last log's term isn't the current term, then according to raft's commit role, leader only commit the log entry with the same term as current term, then leader will never commit
				// It' better to be done after setting nextIndex, to send the newLog to other peers immediately.
				// No neccessary in lab3b, because there is retry and lab2b's index will be wrong. But in real-world, it's better to do so.
				// rf.Log = append(rf.Log, Log{
				// 	Term:  rf.CurrentTerm,
				// 	Index: len(rf.Log),
				// 	Cmd:   -9999, // empty log entry
				// })

				rf.persist()

				rf._sendHeartBeats()
			}

		}
		cond.L.Unlock()

	}()
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	// Leader sendHeartBeats ticking
	go func() {
		for !rf.killed() {
			go rf.sendHeartBeats()
			time.Sleep(HeartBeatInterval)
		}
	}()

	// Follower starts leader election periodically if leader is stale
	for !rf.killed() {
		// Your code here to check if a leader election should
		// be started and to randomize sleeping time using
		// time.Sleep().
		electionTimeout := RandElectionTimeout()
		start := time.Now()
		time.Sleep(electionTimeout)
		// start new election
		rf.mu.Lock()
		if rf.lastLive.Before(start) && rf.role != Leader {
			DPrintf("ticking, server: %v, role: %v, term: %v, start election", rf.me, rf.role, rf.CurrentTerm)
			go rf.kickOffLeaderElection(electionTimeout)

		}
		rf.mu.Unlock()
	}
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	// 2A
	rf.VotedFor = -1
	rf.role = Follower
	rf.lastLive = time.Now()
	rf.CurrentTerm = 0
	rf.commitIndex = -1

	// 2B
	rf.applyCh = applyCh

	// 2C
	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))
	for i := range rf.nextIndex {
		// initialize to leader last log index + 1
		rf.nextIndex[i] = len(rf.Log)
	}
	for i := range rf.nextIndex {
		// initialize to -1
		rf.matchIndex[i] = -1
	}

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	DPrintf("follower: %v, Candicate: %v, leader: %v", Follower, Candicate, Leader)
	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}
