package models

type Order struct {
	id     int
	name   string
	Sender *Sender
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
