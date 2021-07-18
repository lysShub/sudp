// key-value store

package ioer

// 没接收一个数据包就需要从Map(pool)中进行一次查询
// 通过两个双向链表分别存储Map中对应key和value实现对应的功能
// {key_list中的值是升序的, 查询采用二分法？}
