# populator-api-go
This application provides an example API for querying a view model wherein application data has been unified with DCF metadata

Example routes include
- `/data/{number}` Returns up to the desired number of data items and their confidence score
- `/data/count` Returns the total count of data items in the database
- `/data/{id}/annotations` Returns the annotations for a given data item, indicated by its ID
