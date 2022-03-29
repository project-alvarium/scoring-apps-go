# subscriber-go
A DCF subscriber application written in Go.

The role of this particular application is to subscribe to DCF annotations via some pub/sub mechanism and then persist them for later processing,
such as score calculation. Events are currently interpreted as a graph and persisted into [ArangoDB](https://www.arangodb.com/). Determination for
where the event fits in the overall graph comes from the `Action` property on the `Annotation`.

## Action values and their handling ##

**1.) Create**
Create indicates a new piece of data, and the annotations included in this event are relevant to the creation. This indicates the need to persist
a new root vertex.

**2.) Mutate**
Mutate indicates some piece of data being changed, or perhaps versioned. This handler has a special role in that it needs to create a lineage from
the old data to the new and then link the included annotations to the new piece of data.

**3.) Transit**
Transit indicates some piece of data has been received from another Alvarium-enabled service.

**4.) Publish**
Publish indicates we are about to publish a piece of data to another service that is not Alvarium-enabled. You might use this to attest to how data
was handled in its original bounded context, prior to being disseminated.

