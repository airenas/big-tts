include .env

data/mongo:
	mkdir -p $@
data/rabbitmq: 
	mkdir -p $@

init: data/mongo data/rabbitmq

mongosh:
	mongosh --port $(MONGO_EXT_PORT) -u $(MONGO_USER) -p $(MONGO_PASS)

clean:
	rm -rf data
