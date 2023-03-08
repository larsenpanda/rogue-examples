# Some examples that have helped me with various things.

- schema_registry.go
  - This is a schema registry example in franz-go that serializes an avro payload and produces it to Confluent Cloud (SASL_PLAIN/SSL) with Confluent Cloud Schema Registry (BASIC AUTH/SSL). It's particularly helpful when you want to test migrations. This was where I started: https://github.com/twmb/franz-go/tree/master/examples
- docker-compose-console-sasl.yaml
  - This is a docker compose example that stands up a local instance of Redpanda with SASL-SCRAM-512 enabled, an admin/superuser created with specific password and Console connecting as a client in that context. Particularly helpful to test client applications currently running against MSK
