## 问题
primary/secondary write 失败，会回滚吗?
- 不会，可能存在不同的view，GFS就是这么设计的

split brain=> caused by network partition