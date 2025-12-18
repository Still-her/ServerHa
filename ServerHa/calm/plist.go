package calm

import (
	"errors"
	"fmt"
	"sync"
)

var Default []*List

type ElemType interface {
}

//实现接口方法

// 结点
type Node struct {
	Data ElemType
	Pre  *Node
	Next *Node
}

// 链表
type List struct {
	//Name  string
	First *Node
	Last  *Node
	Size  int
	lock  *sync.Mutex
}

func init() {
	for i := 0; i < 10; i++ {
		Default = append(Default, CreateList())
	}
	// for _, ac := range config.Instance.Acnode {
	// 	Default = append(Default, CreateList(ac.Point))
	// }
}

// 工厂函数
func CreateList() *List {
	s := new(Node)
	s.Next, s.Pre = s, s
	lock := new(sync.Mutex)
	return &List{s, s, 0, lock}
}

// 清空队列函数
func (list *List) InvalList() {

	list.lock.Lock()
	defer list.lock.Unlock()

	list.Last = list.First
	list.First.Pre = list.Last
	list.Last.Next = list.First
	list.Size = 0
}

// 尾插法
func (list *List) PushBack(x ElemType) {

	list.lock.Lock()
	defer list.lock.Unlock()

	s := new(Node)
	s.Data = x
	list.Last.Next = s
	s.Pre = list.Last

	list.Last = s
	list.Last.Next = list.First
	list.First.Pre = list.Last
	list.Size++
}

// 头插法
func (list *List) PushFront(x ElemType) {

	list.lock.Lock()
	defer list.lock.Unlock()

	s := new(Node)
	s.Data = x
	s.Next = list.First.Next
	list.First.Next.Pre = s

	list.First.Next = s
	s.Pre = list.First
	if list.Size == 0 {
		list.Last = s
	}
	list.Size++
}

// 尾删法
func (list *List) PopBack() bool {

	if list.IsEmpty() {
		return false
	}

	list.lock.Lock()
	defer list.lock.Unlock()

	s := list.Last.Pre //找到最后一个节点的前驱
	s.Next = list.First
	list.Last = s
	list.Size--
	return true
}

// 头删法
func (list *List) PopFront() bool {
	if list.IsEmpty() {
		return false
	}

	list.lock.Lock()
	defer list.lock.Unlock()

	s := list.First.Next //找到第一个节点
	list.First.Next = s.Next
	s.Next.Pre = list.First
	if list.Size == 1 {
		list.Last = list.First
	}
	list.Size--
	return true
}

// 查找指定元素
func (list *List) Find(x ElemType) *Node {

	list.lock.Lock()
	defer list.lock.Unlock()

	s := list.First.Next
	for s != list.First {
		if x == s.Data {
			return s
		} else {
			s = s.Next
		}
	}
	return nil
}

func (list *List) GetFirst() *Node {

	list.lock.Lock()
	defer list.lock.Unlock()

	return list.First.Next
}

func (list *List) GetSize() int {

	list.lock.Lock()
	defer list.lock.Unlock()

	return list.Size
}

func (list *List) Getlimit(n int) []interface{} {
	var i int
	var et []interface{}
	if list.IsEmpty() {
		return et
	}

	list.lock.Lock()
	defer list.lock.Unlock()

	s := list.First.Next
	for s != list.First && i < n {
		i++
		et = append(et, s.Data)
		s = s.Next
	}
	return et
}

func (list *List) Getpagelimit(page int, limit int) []interface{} {
	var i int
	var et []interface{}
	if list.IsEmpty() {
		return et
	}

	list.lock.Lock()
	defer list.lock.Unlock()

	if list.Size < (page-1)*limit {
		return et
	}
	s := list.First.Next
	for s != list.First && i < (page*limit) {
		i++
		if i > (page-1)*limit {
			et = append(et, s.Data)
		}
		s = s.Next
	}
	return et
}

// 按值删除结点
func (list *List) DeleteVal(x ElemType) bool {

	s := list.Find(x)

	list.lock.Lock()
	defer list.lock.Unlock()

	if s != nil {
		s.Pre.Next = s.Next
		s.Next.Pre = s.Pre
		list.Size--
		//如果删除的是最后一个结点
		if s == list.Last {
			list.Last = s.Pre
		}
		return true
	}
	return false
}

// 把值为x的元素的值修改为y
func (list *List) Modify(x, y ElemType) bool {
	s := list.Find(x)

	list.lock.Lock()
	defer list.lock.Unlock()

	if s != nil {
		s.Data = y
		return true
	}
	return false
}

// 判断链表是否为空
func (list *List) IsEmpty() bool {

	list.lock.Lock()
	defer list.lock.Unlock()

	return list.Size == 0
}

// 反转链表
// 保留第一个结点，将剩余的结点游离出来，然后依次头插到保留的结点中
func (list *List) Reverse() {

	list.lock.Lock()
	defer list.lock.Unlock()

	if list.Size > 1 {
		s := list.First.Next
		p := s.Next
		s.Next = list.First //第一个结点逆置后成为最后一个结点
		list.Last = s

		for p != list.First {
			s = p
			p = p.Next

			s.Next = list.First.Next
			list.First.Next.Pre = s

			s.Pre = list.First
			list.First.Next = s
		}
	}
}

// 打印链表
func (list *List) Print() error {
	if list.IsEmpty() {
		return errors.New("this is an empty list")
	}

	list.lock.Lock()
	defer list.lock.Unlock()

	s := list.First.Next
	for s != list.First {
		fmt.Printf("%v  ", s.Data)
		s = s.Next
	}
	return nil
}

// 保留
func get(params []interface{}) []int {
	var stringSlice []int
	for _, param := range params {
		stringSlice = append(stringSlice, param.(int))
	}
	return stringSlice
}
