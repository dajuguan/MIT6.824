# 核心问题
## How to avoid split votes and ensure only 1 leader is elected?
- randomized re-election
- 一个节点一个term只能有一票，所以只有两种情况需要set votedFor
    - 收到term>currentTerm，reset votedFor为null
    - grantVote时才set VotedFor为投给的节点

## How to ensure the leader store all commited log entries?
确保candidate的log至少和多数人的log一样新=> 如何定义two logs哪个更up to date?
- term更大
- log 更长

# leader election
## 初始都是follower
- if election timeout && (no AppendEntries RPC || granting vote to candidates)
    - start election
        - Increament currentTerm/Vote for self => transite to candicate
        - Reset election timer
        - Vote for self, Send Request vote to all other servers
    - select如下情形之一
        - votes received from majority servers: become leader
        - AppendEntries RPC received from new leader
            - term >= current term => convert to follower; else rejects
        - timeout elapsed with no winner:
            - timeouts, start new election

- 收到AppendEntries
    - stale term, rejects
    - larger term, update current term
    - remain as follower

## 成为leader后
- repeatedly sending empty AppendEntries RPC
- AppendEntries RPC received && term > currentTerm
    - convert to follower

## Candidate
- 收到AppendEntries
    - stale term, rejects
    - larger term, revert to follower
- 收到Request Votes
    - vote at most one candidate in a given term

## 核心技术问题，怎么保证状态转换成功后，cancel调正在进行的任务
- timeout 与定时任务处理
- heartbeat与election开不同线程处理吗？
- appendRPC状态对election的影响怎么处理
- 何时reset vote for
    - 成为candidate: voted for 设置为self
    - 成为leader: 不需要重置
    - grant votes: 重置为null
    - appendEntries Request/Response, requestVote, 需要convert为follower时(term>currentTerm)
        - requestVote&appendEntriesResp: leader/follower收到更高的term, 设置currentTerm=term后，voted for设置为null
            - 必须先设置currentTerm=term, 防止针对过去的currentTerm重复投票
        - appendEntrieRequest: 也需要要求term>currentTerm采reset VotedFor,否则term==currentTerm时直接reset可能导致splitVote
            - 因为如果此前投票给leader了，此时leaderRPC消息来了，term==currentTerm, reset VoteFor
                - 如果electionTimeout还没结束，此时又收到其他节点的term==currentTerm，由于votedFor为null(不阻止投票给其他节点),此时有可能对其他节点的同一个term重复投票
- 何时reset timer：用时钟转动的思路去想
    1.receive appendEntries with term >=currentTerm
        - 注意不像votedFor, term ==currentTerm 时也需要reset timer,来防止其他节点再次start election
    2.start election
    3.grant other peer a vote
election resetrction只针对于election，不需要针对requestEntries