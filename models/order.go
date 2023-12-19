package models

import (
	"fmt"
	"time"
)

// 静态变量区
var branchHelpOrderNum = 0
var branchHelpOrderQueue = &OrderQueue{}

type Order struct {
	id       int
	name     string
	Sender   *Sender
	taskName string
	envs     []Env
}

type OrderQueue struct {
	orders []*Order
}

func (oq *OrderQueue) AddOrder(order *Order) {
	oq.orders = append(oq.orders, order)
}

func (oq *OrderQueue) ProcessOrder() *Order {
	if len(oq.orders) == 0 {
		return nil
	}

	order := oq.orders[0]
	oq.orders = oq.orders[1:]

	return order
}

func initOrder(activity *OrderQueue, activityName string, inviteIdName string) {
	for {
		order := activity.ProcessOrder()
		if order == nil {
			time.Sleep(time.Second * time.Duration(3))
			continue
		}

		fmt.Printf("Processing order %d: %s\n", order.id, order.name)
		runTask(&Task{Path: order.taskName, Envs: order.envs}, order.Sender)
		order.Sender.Reply(fmt.Sprintf("订单ID:%d已完成", order.id))

	}
}
