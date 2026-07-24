package database

// Database defines all of the db operation
type Database interface {
	CountAll() (int, error)                                 //Get db records count
	DeviceTokenByKey(key string) (string, error)            //Get specified device's token
	SaveDeviceTokenByKey(key, token string) (string, error) //Create or update specified devices's token
	SaveMessage(message *Message) error                     //Persist pushed message
	DeleteDeviceByKey(key string) error                     //Delete specified device
	Close() error                                           //Close the database
}
