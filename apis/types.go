package apis

import (
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

// MongoDBOptions represents options that can be used to configure a Database.
type MongoDBOptions struct {
	// The read concern to use for operations executed on the Database. The default value is nil, which means that
	// the read concern of the client used to configure the Database will be used.
	ReadConcern *readconcern.ReadConcern

	// The write concern to use for operations executed on the Database. The default value is nil, which means that the
	// write concern of the client used to configure the Database will be used.
	WriteConcern *writeconcern.WriteConcern

	// The read preference to use for operations executed on the Database. The default value is nil, which means that
	// the read preference of the client used to configure the Database will be used.
	ReadPreference *readpref.ReadPref

	// The BSON registry to marshal and unmarshal documents for operations executed on the Database. The default value
	// is nil, which means that the registry of the client used to configure the Database will be used.
	Registry *bsoncodec.Registry
}