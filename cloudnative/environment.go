package cloudnative

import "os"

type Environment struct {
}

func NewEnvironment() Environment {
	return Environment{}
}

func (e Environment) Services() string {
	services, ok := os.LookupEnv("VCAP_SERVICES")
	if !ok {
		return "{}"
	}

	return services
}

func (e Environment) Stack() string {
	return os.Getenv("CF_STACK")
}
