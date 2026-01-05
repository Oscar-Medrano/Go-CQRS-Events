package events

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"platzi.com/go/cqrs/models"
)

type NatsEventStore struct {
	conn            *nats.Conn
	feedCreatedSub  *nats.Subscription
	feedCreatedChan chan CreatedFeedMessage
}

func NewNats(url string) (*NatsEventStore, error) {
	options := []nats.Option{
		nats.MaxReconnects(-1),//Reconnect indefinitely
		nats.ReconnectWait(2 * time.Second),//Wait 2 seconds before reconnecting
		nats.PingInterval(30 * time.Second),// Send a ping every 30 seconds
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Printf("NATS: Reconnectado exitosamente a %s\n", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			fmt.Printf("NATS: Conexión cerrada: %s\n", nc.LastError())
		}),
	}
	//Aquí es donde realmente se intenta abrir el socket de red hacia la url
	conn, err := nats.Connect(url, options...)
	if err != nil {
		return nil, fmt.Errorf("error conectando a NATS: %w", err)
	}
	//Verifica si la conexión se estableció correctamente
	if !conn.IsConnected() {
		conn.Close()
		return nil, fmt.Errorf("no se pudo establecer conexión con NATS en %s", url)
	}
	//retorno del struct que implementa EventStore
	store := &NatsEventStore{
		conn: conn,
	}
	return store, nil
}

// Cierra la conexión y libera recursos
func (n *NatsEventStore) Close() error {
	var errs []error
	if n.feedCreatedChan != nil {
		select {
		case <-n.feedCreatedChan:
		default:
			close(n.feedCreatedChan)
		}
	}
	if n.feedCreatedSub != nil {
		if err := n.feedCreatedSub.Unsubscribe(); err != nil {
			errs = append(errs, fmt.Errorf("error al desuscribirse de feedCreated: %w", err))
		}
	}
	if n.conn != nil && !n.conn.IsClosed() {
		n.conn.Close()
	}
	n.feedCreatedSub = nil
	n.feedCreatedChan = nil
	n.conn = nil
	if len(errs) > 0 {
		return fmt.Errorf("errores cerrando NATS event store: %v", errs)
	}
	return nil
}

// encodeMessage serializes a Message into a byte slice using gob encoding
func (n *NatsEventStore) encodeMessage(msg Message) ([]byte, error) {
	b := bytes.Buffer{}
	err := gob.NewEncoder(&b).Encode(msg)
	if err != nil {
		return nil, fmt.Errorf("error encoding message: %w", err)
	}
	return b.Bytes(), nil
}

// PublishCreatedFeed publishes a CreatedFeedMessage to NATS
func (n *NatsEventStore) PublishCreatedFeed(ctx context.Context, feed *models.Feed) error {
	msg := CreatedFeedMessage{
		ID:          feed.ID,
		Title:       feed.Title,
		Description: feed.Description,
		CreatedAt:   feed.CreatedAt,
	}
	data, err := n.encodeMessage(msg)
	if err != nil {
		return err
	}
	return n.conn.Publish(msg.Type(), data)
}

// decodeMessage deserializes a byte slice into the provided Message using gob decoding
func (n *NatsEventStore) decodeMessage(data []byte, m interface{}) error {
	b := bytes.Buffer{}
	b.Write(data)
	return gob.NewDecoder(&b).Decode(m)
}
// OnCreatedFeed sets up a subscription to listen for CreatedFeedMessage events on callback style
func (n *NatsEventStore) OnCreatedFeed(f func(CreatedFeedMessage)) (err error) {
	msg := CreatedFeedMessage{}
	n.feedCreatedSub, err = n.conn.Subscribe(msg.Type(), func(m *nats.Msg) {
		n.decodeMessage(m.Data, &msg)
		f(msg)
	})
	return err
}

// SubscribeCreatedFeed sets up a subscription to listen for CreatedFeedMessage events and returns a channel
func (n *NatsEventStore) SubscribeCreatedFeed(ctx context.Context) (<-chan CreatedFeedMessage, error) {
	msg := CreatedFeedMessage{}
	n.feedCreatedChan = make(chan CreatedFeedMessage, 64)
	ch := make(chan *nats.Msg, 64)
	var err error
	n.feedCreatedSub, err = n.conn.ChanSubscribe(msg.Type(), ch)
	if err != nil {
		return nil, err
	}
	go func() {
    	for m := range ch {
        	var msg CreatedFeedMessage  // Nueva instancia en cada iteración
        	n.decodeMessage(m.Data, &msg)
        	n.feedCreatedChan <- msg
    	}
	}()
	return (<-chan CreatedFeedMessage)(n.feedCreatedChan), nil
}