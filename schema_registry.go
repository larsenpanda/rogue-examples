package main

import (
	"context"
	"flag"
	"fmt"
	"crypto/tls"
	"net"
	"os"
	"strings"
	"time"

	"github.com/hamba/avro"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sr"
	"github.com/twmb/franz-go/pkg/sasl/plain"
)

var (
	seedBrokers = flag.String("brokers", "pkc-2396y.us-east-1.aws.confluent.cloud:9092", "comma delimited list of seed brokers")
	topic       = flag.String("topic", "ermagerd", "topic to produce to and consume from")
	user    = flag.String("user", "LJ6TEJ23HMPJDJXC", "kafka user")
	pass    = flag.String("pass", "D2DYqgSv1LHee1xkn1SNvN0hncv/gDo4suysnGwW3OfudtvUOEkSUBg0xzp3nSl7", "kafka pass")
	registry    = flag.String("registry", "https://psrc-gk071.us-east-2.aws.confluent.cloud", "schema registry port to talk to")
	sr_user = flag.String("sr_user", "MEPVB56MF4HPMZSV", "schema user")
	sr_pass = flag.String("sr_pass", "R/wIorBEE+qdkgFjQKnS/CkJEbWMzRn5gHqxPgOrmtPN3ml0wgqzZh8gq6u3HJPO", "schema password")
)

func die(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

func maybeDie(err error, msg string, args ...interface{}) {
	if err != nil {
		die(msg, args...)
	}
}

var schemaText = `{
	"type": "record",
	"name": "simple",
	"namespace": "org.hamba.avro",
	"fields" : [
		{"name": "a", "type": "long"},
		{"name": "b", "type": "string"}
	]
}`

type example struct {
	A int64  `avro:"a"`
	B string `avro:"b"`
}

func main() {
	flag.Parse()
	tlsDialer := &tls.Dialer{NetDialer: &net.Dialer{Timeout: 10 * time.Second}}
	// Ensure our schema is registered.
	rcl, err := sr.NewClient(sr.URLs(*registry), sr.BasicAuth(*sr_user,*sr_pass))
	maybeDie(err, "unable to create schema registry client: %v", err)
	ss, err := rcl.CreateSchema(context.Background(), *topic+"-value", sr.Schema{
		Schema: schemaText,
		Type:   sr.TypeAvro,
	})
	maybeDie(err, "unable to create avro schema: %v", err)
	fmt.Printf("created or reusing schema subject %q version %d id %d\n", ss.Subject, ss.Version, ss.ID)

	// Setup our serializer / deserializer.
	avroSchema, err := avro.Parse(schemaText)
	maybeDie(err, "unable to parse avro schema: %v", err)
	var serde sr.Serde
	serde.Register(
		ss.ID,
		example{},
		sr.EncodeFn(func(v interface{}) ([]byte, error) {
			return avro.Marshal(avroSchema, v)
		}),
		sr.DecodeFn(func(b []byte, v interface{}) error {
			return avro.Unmarshal(avroSchema, b, v)
		}),
	)

	// Loop producing & consuming.
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(strings.Split(*seedBrokers, ",")...),
		kgo.DefaultProduceTopic(*topic),
		kgo.ConsumeTopics(*topic),
		kgo.SASL(plain.Auth{
			User: *user,
			Pass: *pass,
		}.AsMechanism()),
		kgo.Dialer(tlsDialer.DialContext),
	)
	maybeDie(err, "unable to init kgo client: %v", err)
	for {
		cl.Produce(
			context.Background(),
			&kgo.Record{
				Value: serde.MustEncode(example{
					A: time.Now().Unix(),
					B: "hello",
				}),
			},
			func(r *kgo.Record, err error) {
				maybeDie(err, "unable to produce: %v", err)
				fmt.Printf("Produced simple record, value bytes: %x\n", r.Value)
			},
		)

		fs := cl.PollFetches(context.Background())
		fs.EachRecord(func(r *kgo.Record) {
			var ex example
			err := serde.Decode(r.Value, &ex)
			maybeDie(err, "unable to decode record value: %v")

			fmt.Printf("Consumed example: %+v, sleeping 1s\n", ex)
		})
		time.Sleep(time.Second)
	}
}
