package database

type DBManager interface {
	GetMongoDB() *MongodbHandler
	Stop()
	IsHealthy() bool
}
