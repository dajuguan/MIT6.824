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
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.824/labgob"
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
	currentTerm int
	votedFor    int
	// log []
	commitIndex int
	lastApplied int
	lastTerm    int
	// nextIndex[]
	// matchIndex []
	role     Role
	lastLive time.Time
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (2A).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	term = rf.currentTerm
	isleader = rf.role == Leader
	return term, isleader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
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
	// entries[]
	LeaderCommit int
}

func (r RequestAppendEntry) String() string {
	return fmt.Sprintf("term %v, leaderId:%v, prevLogIndex:%v, prevLogTerm: %v", r.Term, r.LeaderId, r.PrevLogIndex, r.PrevLogTerm)
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (rf *Raft) sendAppendEntries(server int, args *RequestAppendEntry, reply *AppendEntriesReply) {
	ok := rf.peers[server].Call("Raft.RequestAppendEntries", args, reply)
	// DPrintf("leader %v, RequestAppendEntries reply from server: %v, success: %v", rf.me, server, reply.Success)

	rf.mu.Lock()
	defer rf.mu.Unlock()
	if ok && reply.Term > rf.currentTerm {
		rf.currentTerm = reply.Term
		rf.role = Follower
		rf.votedFor = -1
	}
}

func (rf *Raft) RequestAppendEntries(args *RequestAppendEntry, reply *AppendEntriesReply) {
	// election timeouts, reset timer
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Success = false
	reply.Term = rf.currentTerm

	if args.Term < rf.currentTerm {
		return
	}

	// transition to follower if term >= currentTerm
	rf.currentTerm = args.Term
	rf.role = Follower

	rf.lastLive = time.Now()
	rf.lastTerm = args.Term
	rf.lastApplied = args.PrevLogIndex
	// ensure only vote once for each term
	if args.Term > rf.currentTerm {
		rf.votedFor = -1
	}

	reply.Success = true
}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

func (r RequestVoteArgs) String() string {
	return fmt.Sprintf("term:%v,id:%v,lastLogIndex:%v,lastLogTerm:%v", r.Term, r.CandidateId, r.LastLogIndex, r.LastLogTerm)
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
	DPrintf("server %v, get requestVote: %v", rf.me, args)
	reply.VoteGranted = false

	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.currentTerm
	if args.Term < rf.currentTerm {
		return
	}

	// transition to follower,  reset currentTerm and votedFor. 缺一不可!
	// currentTerm must be updated to ensure one peer is only allowed to vote once for each term
	if args.Term > rf.currentTerm {
		rf.role = Follower
		// make sure kickoff next election use latest term
		rf.currentTerm = args.Term
		rf.votedFor = -1
	}

	// lastLive should only be updated after granting vote, or the current server will issue leader election again
	if rf.votedFor == -1 || rf.votedFor == args.CandidateId {
		// candidate's log must be at least update to date with the majority.
		// shouldn't compare rf.currentTerm!
		if args.LastLogTerm > rf.lastTerm {
			reply.VoteGranted = true
			rf.votedFor = args.CandidateId
			rf.lastLive = time.Now()
			return
		}
		if args.LastLogTerm == rf.lastTerm {
			if args.LastLogIndex >= rf.lastApplied {
				reply.VoteGranted = true
				rf.votedFor = args.CandidateId
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
	DPrintf("send requestVote from: %v, to :%v, term: %v", rf.me, server, rf.currentTerm)
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
	isLeader := true

	// Your code here (2B).

	return index, term, isLeader
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
	if rf.role != Leader {
		rf.mu.Unlock()
		return
	}
	rf.lastLive = time.Now()
	rf.lastApplied += 1
	rf.lastTerm = rf.currentTerm

	// send heartbeat
	entry := RequestAppendEntry{
		Term:         rf.currentTerm,
		LeaderId:     rf.me,
		PrevLogIndex: rf.lastApplied,
		PrevLogTerm:  rf.lastTerm,
	}
	rf.mu.Unlock()

	// DPrintf("leader: %v, request append entries: %v", rf.me, entry)
	for server := range rf.peers {
		if server == rf.me {
			continue
		}
		// use goroutine for each rpc to avoid blocking
		server := server
		go func() {
			reply := AppendEntriesReply{}
			rf.sendAppendEntries(server, &entry, &reply)
		}()

	}
}

// Mustn't use seed or it'll generate the same random number every time
func RandElectionTimeout() time.Duration {
	randVal := rand.Float32()*3 + 1.2
	duration := int64(float32(HeartBeatInterval) * randVal)
	return time.Duration(duration)
}

func (rf *Raft) kickOffLeaderElection() {
	rf.mu.Lock()
	rf.role = Candicate
	rf.currentTerm += 1
	rf.votedFor = rf.me
	rf.lastLive = time.Now()
	term := rf.currentTerm
	rf.mu.Unlock()

	go func() {
		// send Rpc request to other servers
		var mu sync.Mutex
		// cond share the same lock with mu cause they control the same shared variables
		cond := sync.NewCond(&mu)
		totalVotes := 1
		finished := 1

		requestVotes := RequestVoteArgs{
			Term:         term,
			CandidateId:  rf.me,
			LastLogIndex: rf.lastApplied,
			LastLogTerm:  rf.lastTerm,
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
			DPrintf("peer: %v, selected as leader", rf.me)
			rf.mu.Lock()
			defer rf.mu.Unlock()
			if rf.role == Candicate && rf.currentTerm == term {
				rf.role = Leader
				rf.votedFor = -1
				// must use go routine to avoid deadlock
				go rf.sendHeartBeats()
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
			DPrintf("ticking, server: %v, role: %v, term: %v, start election", rf.me, rf.role, rf.currentTerm)
			go rf.kickOffLeaderElection()

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
	rf.votedFor = -1
	rf.role = Follower
	rf.lastLive = time.Now()
	rf.currentTerm = 0

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	DPrintf("follower: %v, Candicate: %v, leader: %v", Follower, Candicate, Leader)
	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}
