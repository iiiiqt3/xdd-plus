package comm

import (
	"fmt"
	"log"

	"github.com/astaxie/beego"
	"github.com/streadway/amqp"
)

// 全局，变量用于存储 RabbitMQ 连接和通道
var (
	conn    *amqp.Connection
	channel *amqp.Channel
)

// 初始化 RabbitMQ 连接
func RabbitMQSetup() {
	rabbitmqEnabled, err := beego.AppConfig.Bool("rabbitmq")
	if err != nil {
		beego.Error("Failed to read rabbitmq config:", err)
		return // 或设置默认值
	}
	if !rabbitmqEnabled {
		return
	}
	// 连接到 RabbitMQ 服务器
	conn, err = amqp.Dial(beego.AppConfig.String("rabbitmqurl"))
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	// log.Println("Connected to RabbitMQ")

	//	 创建通道
	channel, err = conn.Channel()
	if err != nil {
		log.Fatalf("Failed to create channel: %v", err)
	}
	fmt.Println("connect RabbitMQ success")

}

// 推送消息到 RabbitMQ
func PublishRabbitMq(k string, message []byte) {
	// 声明队列
	var err error
	_, err = channel.QueueDeclare(
		k,     // 队列名称
		true,  // 是否持久化 含义​：队列的元数据（如队列名、绑定关系）是否保存到磁盘。true：队列在 RabbitMQ 服务重启后仍然存在。false：队列仅存于内存，重启后丢失（但消息是否持久化取决于发布时的 delivery_mode 参数）。​适用场景​：需长期保留的队列（如订单队列）应设为 true，临时任务队列可设为 false。
		false, // 是否独占
		false, // 是否自动删除 当所有消费者断开连接后，队列是否自动删除。true：最后一个消费者断开后，队列自动删除 false：队列持续存在，直到显式删除或服务重启（若未持久化）
		false, // 是否阻塞
		nil,   // 额外参数
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}
	// 发送消息
	err = channel.Publish(
		"",    // 交换机名称，使用空字符串表示直接将消息发送到队列
		k,     // 队列名称
		false, // 是否强制发送（如果为 true，消息将被发送到所有绑定的队列）
		false, // 是否立即发送（如果为 true，消息将被立即发送，否则将被缓存）
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		},
	)
	if err != nil {
		log.Fatalf("Failed to publish message: %v", err)
	}
	log.Printf("Message sent: %s", message)
}
