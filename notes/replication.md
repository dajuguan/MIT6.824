## what kind failures replication can solve?
- √ fail-stop faults
- x bugs in software 
- x correlated failure: 地震，同意设备商的设备

## how to replicate
- state transfer: send the whole state
- relicated state machine
    - just send operations
    - 一般情况下OP比state更小

## problems to implement replicated state machine
- what state?
    - app state
- primary/backup(P/B) sync
    - non-deterministic events
        - inputs data + interupt(执行到什么指令产生中断)， 99%的情况
            - 由VM监视器转发给P/B,避免数据包不一致（硬件商直接允许）
        - weird instructions (随机数等)
        - multi-cores
    - log entry:
        - instruction number
        - type(上述三种之一)
        - data
- client cut-over切换节点
- anomalies 处理异常
- New relicas

## 如何确保client看到的输出一样？
- VMM采用output rule进行限制
如果主节点向backup节点同步数据时crash了，那么VM监视器(VMM)会阻断向客户端发送输出，直到它确认backup得到了数据

- 切换到backup时，client可能会收到重复的回复(backup先重复执行之前的指令，同时又受到了client的retry)，怎么处理？
    - 可以发送相同的TCP帧，这样在TCP这一层就把packet drop掉，用户端无感 

## 主从节点网络断了，splitBrain，两个节点同时上线
- 设置一个中心化仲裁节点，来确定谁是主节点
    - 设置test and set server，每次节点想请求成为主节点时需要请求该server，能否设置为primary