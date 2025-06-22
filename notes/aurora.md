# History of Amazon cloud
- EBS问题：
    - chain replication需要大量网络带宽
    - EBS不是fault tolerant的
- AZ(Availiability Zone)，将EBS分为不同可用区，在同一可用区进行备份
- RDS(Relational Database Service)异地备份 
    - 备份太慢了，因为异地
    - 数据量超大，网络开销大: 直接发送的是MySQL的transaction日志，通常以page为单位
- Aurora (三个可用区，每个用2个replica)
    - 用自己设计的DB，只发log entry，数据量较小
    - quorum: 6个里面只需要4个备份就行了
- Aurota F.T(fault tolerance) goals
    - Write: 1 dead AZ
    - Read: 1dead AZ + 1 server
    - Transitient slow storage 
    - Fast REPL(replica)
- Autora design
    - 超过一个机器能容纳，则sharding，分为Protection Log
    - 一个机器上分别存不同instance对应的数据，这样一个PG down掉之后可以并发从不同的server复制，这样可以更快地恢复
    - mini transaction: read database要么读一批transaction还没写入之前的状态，要么读写入之后的状态，不会在数据库更新到一半的过程中读

# paper中有意思的思路总结
- DB中transaction和storage处理的过程
- quorum
- DB和storage co-design来实现更好的性能(35X性能提升)
- Implicit information，Amazon工程师重视的F.T.设计原则+网络开销很大

