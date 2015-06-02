# Microservice CQRS Scaleout

I have been spending some time earlier in the week looking at some CQRS topics within the [CQRS Framework](https://github.com/andrewwebber/cqrs), in particular concurrency and scaleablity.

####Write Model Concurrency
We want to have a situation where write model aggregates can be updated in parallel but that individual aggregates still have instance level concurrency checks. For example two users try to rename a file concurrently.

Concurrency at this level is achieved by the storage provider for the aggregate. When an aggregate is saved by the repository, the repository checks if the aggregate instance to be saved is the latest version. If it is not, the repository will throw an error. This means in the example case above, the last user to try to rename the file will receive a REST api concurrency error. You have two choices here, 1. return the concurrency error to the caller or in most cases the framework can put the request back on the queue and it will be retried.  However in many cases a retry is useful, for example 'add comment'. If the aggregate is a comments collection we might be ok with a retry of two people commenting at the same time on the same item. The last commenter will simply have their command retried and the command will mostly succeed. 

It is important that any operations on aggregates only change the internal state of the aggregate, so they can be retried. Side effects, like sending emails, should be done in events - which are the results of aggregates being successfully saved.

####Read Model Concurrency
When an event is pushed onto the event bus it is broadcast to all micro services. This not only includes micro services of different types but multiple instances of the same type of micro service. The CQRS framework is configured so that instances of a micro service compete for queue entries forwarded to the service type. However when two instances are receiving events for the service type the instances do not wait for other while one is processing a message. For example instance 1 and instance 2 of service type A can process messages concurrently. These messages might be related to the same read model and in the worst case the same item level read model. So just like the write model we need to be careful of concurrency issues. 

I created an example to experiment with concurrency issues and after proving these concurrency issues refactored the example with a working fix

https://github.com/andrewwebber/cqrs-scaleout

The sample is a simple publish/subscribe setup where I aim to publish 10 messages and in the read model I want to have a counter of messages processed. I purposely make the read model code buggy and don’t use Couchbase’s incr helper.

Publisher
```golang
for i := 0; i < 10; i++ {
	if err := bus.PublishEvents([]cqrs.VersionedEvent{cqrs.VersionedEvent{
		Version:   i,
		EventType: eventType.String(),
		Event:     scaleout.SampleEvent{"rabbit_TestEvent"}}}); err != nil {
		failOnError(err, "publish event")
	}
}
```

Subscriber
```golang
type DataObject struct {
	Message string
	Count   int
}

func updateValueInCouchbaseErrorProne(message string) error {
	var dataObject DataObject
	dataObjectKey := "cqrs-scaleout:dataobject"
	err := bucket.Get(dataObjectKey, &dataObject)
	if err != nil {
		if !IsNotFoundError(err) {
			return err
		}

		dataObject = DataObject{Message: message, Count: 0}
	}

	dataObject.Count = dataObject.Count + 1
	log.Println(dataObject.Count)
	return bucket.Set(dataObjectKey, 0, &dataObject)
}
```

Every time a messages is read from the queue the app goes to the database and does a simply computer science concurrency bug, namely GET, UPDATE, SET. 
When running multiple instances of the app in some cases the counter in the database does not add up to 10, our expected value. This is obviously because in some cases there is a dirty read.

The easiest way to fix this problem is to use the Couchbase **CAS** (Compare-and-set) feature.

```golang
func updateValueInCouchbaseV2UsingCAS(message string) error {
	var dataObject DataObject
	dataObjectKey := "cqrs-scaleout:dataobject"
	var cas uint64
	err := bucket.Gets(dataObjectKey, &dataObject, &cas)
	if err != nil {
		if !IsNotFoundError(err) {
			return err
		}

		dataObject = DataObject{Message: message, Count: 0}
	}

	dataObject.Count = dataObject.Count + 1
	log.Println(dataObject.Count)
	log.Println(cas)
	return bucket.Cas(dataObjectKey, 0, cas, &dataObject)
}
```

The **CAS** feature allows use to read a value from the database and only update it if the value has not changed since we read from the database. This will return an error to the application when a dirty update might take place.

In this case its ok to return an error because we can tell the CQRS framework to put the request back on the queue and have it retried. Then the retry logic kicks in the read model is eventually, at the end of the test, up to date with the correct counter value.

```golang
case event := <-receiveEventChannel:
	sampleEvent := event.Event.Event.(scaleout.SampleEvent)
	err := updateValueInCouchbaseV2UsingCAS(sampleEvent.Message)
	if err == nil {
		event.ProcessedSuccessfully <- true
	} else {
		log.Println(err)
		// Request should get retryied
		event.ProcessedSuccessfully <- false
	}
```

The goal is obviously to design the system (aggregates and read models) such that the entities are less likely to be concurrently updated as possible. This is where designing how many aggregates or read models we need per business problem improves or degrades performance (concurrency conflicts). For example having an Organisation as an aggregate might be ok because the Organisation might not be updated concurrently by to many users at once. However having a root folder as an aggregate, that maintained all its items in the aggregate, might result in to many concurrency cases where users are sharing a folder and modifying its contents. 
